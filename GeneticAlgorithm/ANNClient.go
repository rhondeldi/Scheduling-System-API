package GeneticAlgorithm

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

// ANNClient communicates with the Python ANN API service.
type ANNClient struct {
	BaseURL    string
	HTTPClient *http.Client
	cache      *annPredictionCache
}

// ─── wire types — match the Python Pydantic models exactly ───────────────────

// SchedulePayload is the format sent to the ANN API (6 days × 24 slots × 3 attributes).
// It serialises to {"week_schedule": [...]}, matching Python's ScheduleData model.
type SchedulePayload struct {
	WeekSchedule [][][]int `json:"week_schedule"`
}

// ScheduleData is kept for historical compatibility with existing GA code.
type ScheduleData = SchedulePayload

type ConstraintPrediction struct {
	InstructorConflict float64 `json:"instructor_conflict"`
	RoomConflict       float64 `json:"room_conflict"`
	NoLunchBreak       float64 `json:"no_lunch_break"`
	LateClasses        float64 `json:"late_classes"`
	ExcessiveHours     float64 `json:"excessive_hours"`
	SaturdayOverload   float64 `json:"saturday_overload"`
}

type ConstraintBatchRequest struct {
	Schedules []ScheduleData `json:"schedules"`
}

type ConstraintBatchResponse struct {
	Predictions      []ConstraintPrediction `json:"predictions"`
	ProcessingTimeMs float64                `json:"processing_time_ms"`
}

type CrossoverPair struct {
	Parent1 ScheduleData `json:"parent1"`
	Parent2 ScheduleData `json:"parent2"`
}

type CrossoverPrediction struct {
	Compatible bool    `json:"compatible"`
	Confidence float64 `json:"confidence"`
}

type CrossoverBatchRequest struct {
	Pairs []CrossoverPair `json:"pairs"`
}

type CrossoverBatchResponse struct {
	Predictions      []CrossoverPrediction `json:"predictions"`
	ProcessingTimeMs float64               `json:"processing_time_ms"`
}

type MutationRequest struct {
	BeforeSchedule ScheduleData `json:"before_schedule"`
	AfterSchedule  ScheduleData `json:"after_schedule"`
	MutationType   string       `json:"mutation_type"`
}

type MutationPrediction struct {
	Label       string  `json:"label"`
	ImprovProb  float64 `json:"improve_prob"`
	NeutralProb float64 `json:"neutral_prob"`
	WorsenProb  float64 `json:"worsen_prob"`
}

type MutationBatchRequest struct {
	Predictions []MutationRequest `json:"predictions"`
}

type MutationBatchResponse struct {
	Predictions      []MutationPrediction `json:"predictions"`
	ProcessingTimeMs float64              `json:"processing_time_ms"`
}

type PreExtractedFeatures struct {
	Features []float64 `json:"features"`
}

type FeatureBatchRequest struct {
	FeatureVectors []PreExtractedFeatures `json:"feature_vectors"`
}

type FitnessBatchPredictionResponse struct {
	Predictions      []FitnessPredictionResponse `json:"predictions"`
	ProcessingTimeMs float64                     `json:"processing_time_ms"`
}

// ─── request types — each wraps SchedulePayload to match the Pydantic schema ─

// fitnessSingleRequest matches Python's FitnessPredictionRequest:
//
//	class FitnessPredictionRequest(BaseModel):
//	    schedule: ScheduleData
type fitnessSingleRequest struct {
	Schedule SchedulePayload `json:"schedule"`
}

// constraintCheckRequest matches Python's ConstraintCheckRequest:
//
//	class ConstraintCheckRequest(BaseModel):
//	    schedule: ScheduleData
type constraintCheckRequest struct {
	Schedule SchedulePayload `json:"schedule"`
}

// batchFitnessItem matches the per-item shape in the batch endpoint:
//
//	{"schedule": {"week_schedule": [...]}}
type batchFitnessItem struct {
	Schedule SchedulePayload `json:"schedule"`
}

// batchFitnessRequest matches Python's batch endpoint outer envelope:
//
//	{"schedules": [...]}
type batchFitnessRequest struct {
	Schedules []batchFitnessItem `json:"schedules"`
}

// crossoverRequest matches Python's CrossoverRecommendationRequest:
//
//	class CrossoverRecommendationRequest(BaseModel):
//	    parent1: ScheduleData
//	    parent2: ScheduleData
//	    parent1_fitness: float
//	    parent2_fitness: float
type crossoverRequest struct {
	Parent1        SchedulePayload `json:"parent1"`
	Parent2        SchedulePayload `json:"parent2"`
	Parent1Fitness float64         `json:"parent1_fitness"`
	Parent2Fitness float64         `json:"parent2_fitness"`
}

// mutationRequest matches Python's /predict/mutation contract.
type mutationRequest struct {
	BeforeSchedule SchedulePayload `json:"before_schedule"`
	AfterSchedule  SchedulePayload `json:"after_schedule"`
	MutationType   string          `json:"mutation_type"`
	BeforeFitness  float64         `json:"before_fitness"`
	AfterFitness   float64         `json:"after_fitness"`
}

// ─── response types ───────────────────────────────────────────────────────────

// FitnessPredictionResponse from the API.
type FitnessPredictionResponse struct {
	PredictedFitness float64 `json:"predicted_fitness"`
	Confidence       float64 `json:"confidence"`
	ProcessingTimeMs float64 `json:"processing_time_ms"`
}

// ConstraintViolation response from the API.
type ConstraintViolation struct {
	Violations       map[string]bool    `json:"violations"`
	ViolationScores  map[string]float64 `json:"violation_scores"`
	ProcessingTimeMs float64            `json:"processing_time_ms"`
}

// CrossoverRecommendation response from the API.
type CrossoverRecommendation struct {
	RecommendedPoints []int     `json:"recommended_points"`
	Probabilities     []float64 `json:"probabilities"`
	ProcessingTimeMs  float64   `json:"processing_time_ms"`
}

// mutationSinglePrediction is the legacy response from /predict/mutation.
type mutationSinglePrediction struct {
	Prediction       string             `json:"prediction"`
	Confidence       float64            `json:"confidence"`
	Probabilities    map[string]float64 `json:"probabilities"`
	ProcessingTimeMs float64            `json:"processing_time_ms"`
}

// HealthResponse from the API.
type HealthResponse struct {
	Status       string          `json:"status"`
	ModelsLoaded map[string]bool `json:"models_loaded"`
	Timestamp    string          `json:"timestamp"`
}

// ─── client ───────────────────────────────────────────────────────────────────

type annPredictionCache struct {
	mu         sync.RWMutex
	fitness    map[uint64]float64
	constraint map[uint64]ConstraintPrediction
	maxSize    int
}

func newANNPredictionCache(maxSize int) *annPredictionCache {
	return &annPredictionCache{
		fitness:    make(map[uint64]float64),
		constraint: make(map[uint64]ConstraintPrediction),
		maxSize:    maxSize,
	}
}

func (c *annPredictionCache) getFitness(hash uint64) (float64, bool) {
	if c == nil {
		return 0, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.fitness[hash]
	return v, ok
}

func (c *annPredictionCache) setFitness(hash uint64, value float64) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.fitness) >= c.maxSize {
		count := 0
		for k := range c.fitness {
			delete(c.fitness, k)
			count++
			if count >= c.maxSize/2 {
				break
			}
		}
	}
	c.fitness[hash] = value
}

func (c *annPredictionCache) getConstraint(hash uint64) (ConstraintPrediction, bool) {
	if c == nil {
		return ConstraintPrediction{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.constraint[hash]
	return v, ok
}

func (c *annPredictionCache) setConstraint(hash uint64, value ConstraintPrediction) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.constraint) >= c.maxSize {
		count := 0
		for k := range c.constraint {
			delete(c.constraint, k)
			count++
			if count >= c.maxSize/2 {
				break
			}
		}
	}
	c.constraint[hash] = value
}

// NewANNClient creates a new ANN API client.
func NewANNClient(baseURL string) *ANNClient {
	return &ANNClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: newANNPredictionCache(10000),
	}
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// fastAPIError is the shape FastAPI returns for 4xx/5xx responses:
//
//	{"detail": "message"} or {"detail": [{"msg": "...", ...}]}
//
// We only need the raw field for error reporting.
type fastAPIError struct {
	Detail json.RawMessage `json:"detail"`
}

// readAPIError reads the response body of a non-2xx response and returns a
// formatted error that includes the HTTP status, the URL, and the FastAPI
// detail message when present.  This centralises all error-body handling so
// each method only needs to check the status code.
func readAPIError(resp *http.Response, url string) error {
	rawBody, _ := io.ReadAll(resp.Body)

	// Try to extract the FastAPI detail field for a clean message.
	var apiErr fastAPIError
	if json.Unmarshal(rawBody, &apiErr) == nil && len(apiErr.Detail) > 0 {
		// Detail may be a plain string or an array of validation errors.
		// Either way, surface it verbatim.
		detail := strings.Trim(string(apiErr.Detail), `"`)
		return fmt.Errorf("%s returned HTTP %d: %s", url, resp.StatusCode, detail)
	}

	// Fallback: include the raw body (truncated to avoid log spam).
	body := string(rawBody)
	if len(body) > 300 {
		body = body[:300] + "…"
	}
	return fmt.Errorf("%s returned HTTP %d: %s", url, resp.StatusCode, body)
}

// post marshals reqBody to JSON and POSTs it to url, returning the raw response
// body bytes on success (2xx).  On any network or non-2xx error it returns a
// descriptive error that includes the URL and (for FastAPI responses) the
// detail message.
func (client *ANNClient) post(url string, reqBody interface{}) ([]byte, error) {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request for %s: %w", url, err)
	}

	resp, err := client.HTTPClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("POST %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %w", url, err)
	}
	return body, nil
}

// ─── public API ───────────────────────────────────────────────────────────────

func hashWeekSchedule(sched [][][]int) uint64 {
	h := fnv.New64a()
	var b [4]byte
	for _, day := range sched {
		for _, slot := range day {
			for _, val := range slot {
				binary.LittleEndian.PutUint32(b[:], uint32(val))
				_, _ = h.Write(b[:])
			}
		}
	}
	return h.Sum64()
}

// HealthCheck verifies that the ANN API service is reachable, reports itself
// healthy, and has at least the fitness model loaded (since the API always
// returns status "healthy" regardless of model state).
func (client *ANNClient) HealthCheck() error {
	url := client.BaseURL + "/health"
	resp, err := client.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("GET %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return readAPIError(resp, url)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to parse health response from %s: %w", url, err)
	}

	if health.Status != "healthy" {
		return fmt.Errorf("ANN API at %s is not healthy (status=%q)", url, health.Status)
	}

	// The Python service always returns "healthy" even when no models are loaded.
	// Check the fitness model explicitly since it is required for GA operation.
	if loaded, ok := health.ModelsLoaded["fitness_predictor"]; ok && !loaded {
		return fmt.Errorf(
			"ANN API at %s is up but fitness_predictor model is not loaded (models_loaded=%v)",
			url, health.ModelsLoaded,
		)
	}

	return nil
}

// PredictFitness calls POST /predict/fitness for a single schedule.
// Input: weekSchedule is [6 days][24 slots][3 attributes].
//
// FIX: previously sent {"week_schedule": [...]} — the API wraps ScheduleData
// inside a "schedule" key, so the correct payload is
//
//	{"schedule": {"week_schedule": [...]}}
func (client *ANNClient) PredictFitness(weekSchedule [][][]int) (float64, error) {
	url := client.BaseURL + "/predict/fitness"

	req := fitnessSingleRequest{
		Schedule: SchedulePayload{WeekSchedule: weekSchedule},
	}

	body, err := client.post(url, req)
	if err != nil {
		return 0, err
	}

	var pred FitnessPredictionResponse
	if err := json.Unmarshal(body, &pred); err != nil {
		return 0, fmt.Errorf("failed to parse fitness response from %s: %w", url, err)
	}
	return pred.PredictedFitness, nil
}

// PredictFitnessBatch predicts fitness for multiple schedules in one call via
// POST /predict/fitness/batch.  Returns one score per input schedule in order.
func (client *ANNClient) PredictFitnessBatch(schedules [][][][]int) ([]float64, error) {
	return client.BatchPredictFitness(schedules)
}

func (client *ANNClient) callBatchFitnessAPI(schedules [][][][]int) ([]float64, error) {
	if len(schedules) == 0 {
		return nil, nil
	}

	url := client.BaseURL + "/predict/fitness/batch"

	items := make([]batchFitnessItem, len(schedules))
	for i, ws := range schedules {
		items[i] = batchFitnessItem{Schedule: SchedulePayload{WeekSchedule: ws}}
	}

	body, err := client.post(url, batchFitnessRequest{Schedules: items})
	if err != nil {
		return nil, err
	}

	var preds []FitnessPredictionResponse
	if err := json.Unmarshal(body, &preds); err != nil {
		return nil, fmt.Errorf("failed to parse batch fitness response from %s: %w", url, err)
	}

	results := make([]float64, len(preds))
	for i, p := range preds {
		results[i] = p.PredictedFitness
	}
	return results, nil
}

func (client *ANNClient) callBatchFitnessFeatureAPI(schedules [][][][]int) ([]float64, error) {
	if len(schedules) == 0 {
		return nil, nil
	}

	url := client.BaseURL + "/predict/fitness/batch/preextracted"
	vectors := make([]PreExtractedFeatures, len(schedules))
	for i, sched := range schedules {
		vectors[i] = PreExtractedFeatures{Features: extractFitnessFeaturesFromPayload(sched)}
	}

	body, err := client.post(url, FeatureBatchRequest{FeatureVectors: vectors})
	if err != nil {
		return nil, err
	}

	var response FitnessBatchPredictionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse pre-extracted fitness response from %s: %w", url, err)
	}

	results := make([]float64, len(response.Predictions))
	for i, p := range response.Predictions {
		results[i] = p.PredictedFitness
	}
	return results, nil
}

// BatchPredictFitness is an alias for PredictFitnessBatch kept for
// compatibility with existing GA call-sites.
func (client *ANNClient) BatchPredictFitness(schedules [][][][]int) ([]float64, error) {
	if len(schedules) == 0 {
		return nil, nil
	}

	cached := make([]float64, len(schedules))
	needsPredict := make([]int, 0)

	for i, sched := range schedules {
		hash := hashWeekSchedule(sched)
		if val, ok := client.cache.getFitness(hash); ok {
			cached[i] = val
			continue
		}
		needsPredict = append(needsPredict, i)
	}

	if len(needsPredict) == 0 {
		return cached, nil
	}

	uncachedScheds := make([][][][]int, len(needsPredict))
	for i, idx := range needsPredict {
		uncachedScheds[i] = schedules[idx]
	}

	predictions, err := client.callBatchFitnessFeatureAPI(uncachedScheds)
	if err != nil {
		predictions, err = client.callBatchFitnessAPI(uncachedScheds)
	}
	if err != nil {
		return nil, err
	}
	if len(predictions) != len(uncachedScheds) {
		return nil, fmt.Errorf("fitness batch response length mismatch: got %d, want %d", len(predictions), len(uncachedScheds))
	}

	for i, idx := range needsPredict {
		hash := hashWeekSchedule(schedules[idx])
		client.cache.setFitness(hash, predictions[i])
		cached[idx] = predictions[i]
	}

	return cached, nil
}

// CheckConstraints calls POST /predict/constraints for a single schedule.
//
// FIX: previously sent {"week_schedule": [...]} — the correct payload is
//
//	{"schedule": {"week_schedule": [...]}}
func (client *ANNClient) CheckConstraints(weekSchedule [][][]int) (*ConstraintViolation, error) {
	url := client.BaseURL + "/predict/constraints"

	req := constraintCheckRequest{
		Schedule: SchedulePayload{WeekSchedule: weekSchedule},
	}

	body, err := client.post(url, req)
	if err != nil {
		return nil, err
	}

	var result ConstraintViolation
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse constraint response from %s: %w", url, err)
	}
	return &result, nil
}

func (client *ANNClient) BatchPredictConstraints(schedules [][][][]int) ([]ConstraintPrediction, error) {
	if len(schedules) == 0 {
		return nil, nil
	}

	cached := make([]ConstraintPrediction, len(schedules))
	needsPredict := make([]int, 0)

	for i, sched := range schedules {
		hash := hashWeekSchedule(sched)
		if val, ok := client.cache.getConstraint(hash); ok {
			cached[i] = val
			continue
		}
		needsPredict = append(needsPredict, i)
	}

	if len(needsPredict) == 0 {
		return cached, nil
	}

	reqSchedules := make([]ScheduleData, len(needsPredict))
	for i, idx := range needsPredict {
		reqSchedules[i] = ScheduleData{WeekSchedule: schedules[idx]}
	}

	url := client.BaseURL + "/predict/constraint/batch"
	body, err := client.post(url, ConstraintBatchRequest{Schedules: reqSchedules})
	if err != nil {
		return nil, err
	}

	var response ConstraintBatchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse constraint batch response from %s: %w", url, err)
	}
	if len(response.Predictions) != len(reqSchedules) {
		return nil, fmt.Errorf("constraint batch response length mismatch: got %d, want %d", len(response.Predictions), len(reqSchedules))
	}

	for i, idx := range needsPredict {
		hash := hashWeekSchedule(schedules[idx])
		client.cache.setConstraint(hash, response.Predictions[i])
		cached[idx] = response.Predictions[i]
	}
	return cached, nil
}

// RecommendCrossover calls POST /recommend/crossover.
// Parent schedules are [6 days][24 slots][3 attributes].
func (client *ANNClient) RecommendCrossover(
	parent1, parent2 [][][]int,
	fitness1, fitness2 float64,
) (*CrossoverRecommendation, error) {
	url := client.BaseURL + "/recommend/crossover"

	req := crossoverRequest{
		Parent1:        SchedulePayload{WeekSchedule: parent1},
		Parent2:        SchedulePayload{WeekSchedule: parent2},
		Parent1Fitness: fitness1,
		Parent2Fitness: fitness2,
	}

	body, err := client.post(url, req)
	if err != nil {
		return nil, err
	}

	var result CrossoverRecommendation
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse crossover response from %s: %w", url, err)
	}
	return &result, nil
}

func (client *ANNClient) BatchPredictCrossover(pairs [][2][][][]int) ([]CrossoverPrediction, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	reqPairs := make([]CrossoverPair, len(pairs))
	for i, pair := range pairs {
		reqPairs[i] = CrossoverPair{
			Parent1: ScheduleData{WeekSchedule: pair[0]},
			Parent2: ScheduleData{WeekSchedule: pair[1]},
		}
	}

	url := client.BaseURL + "/predict/crossover/batch"
	body, err := client.post(url, CrossoverBatchRequest{Pairs: reqPairs})
	if err != nil {
		return nil, err
	}

	var response CrossoverBatchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse crossover batch response from %s: %w", url, err)
	}
	if len(response.Predictions) != len(reqPairs) {
		return nil, fmt.Errorf("crossover batch response length mismatch: got %d, want %d", len(response.Predictions), len(reqPairs))
	}
	return response.Predictions, nil
}

// PredictMutation calls POST /predict/mutation using the post-hoc before/after
// mutation record expected by the retrained model.
func (client *ANNClient) PredictMutation(
	beforeSchedule SchedulePayload,
	afterSchedule SchedulePayload,
	mutationType string,
	beforeFitness float64,
	afterFitness float64,
) (*mutationSinglePrediction, error) {
	url := client.BaseURL + "/predict/mutation"

	req := mutationRequest{
		BeforeSchedule: beforeSchedule,
		AfterSchedule:  afterSchedule,
		MutationType:   mutationType,
		BeforeFitness:  beforeFitness,
		AfterFitness:   afterFitness,
	}

	body, err := client.post(url, req)
	if err != nil {
		return nil, err
	}

	var result mutationSinglePrediction
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse mutation response from %s: %w", url, err)
	}
	return &result, nil
}

func (client *ANNClient) BatchPredictMutation(requests []MutationRequest) ([]MutationPrediction, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	url := client.BaseURL + "/predict/mutation/batch"
	body, err := client.post(url, MutationBatchRequest{Predictions: requests})
	if err != nil {
		return nil, err
	}

	var response MutationBatchResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse mutation batch response from %s: %w", url, err)
	}
	if len(response.Predictions) != len(requests) {
		return nil, fmt.Errorf("mutation batch response length mismatch: got %d, want %d", len(response.Predictions), len(requests))
	}
	return response.Predictions, nil
}

func extractFitnessFeaturesFromPayload(sched [][][]int) []float64 {
	var week Schedule.WeekTimeTable
	for day := 0; day < len(sched) && day < 6; day++ {
		for slot := 0; slot < len(sched[day]) && slot < 24; slot++ {
			if len(sched[day][slot]) < 3 {
				continue
			}
			_ = week[day][slot].SetSubjectID(uint16(sched[day][slot][0]))
			_ = week[day][slot].SetInstructorID(uint16(sched[day][slot][1]))
			_ = week[day][slot].SetRoomID(uint16(sched[day][slot][2]))
		}
	}
	return extractFitnessFeatures(week)
}

func extractFitnessFeatures(week Schedule.WeekTimeTable) []float64 {
	features := make([]float64, 0, 48)

	dailyHours := make([]float64, 6)
	for day := 0; day < 6; day++ {
		occupied := 0
		for slot := 0; slot < 24; slot++ {
			if week[day][slot].GetSubjectID() > 0 {
				occupied++
			}
		}
		dailyHours[day] = float64(occupied) / 2.0
	}

	features = append(features, dailyHours...)
	total := 0.0
	activeDays := 0.0
	maxDay := 0.0
	for _, h := range dailyHours {
		total += h
		if h > 0 {
			activeDays++
		}
		if h > maxDay {
			maxDay = h
		}
	}
	variance := 0.0
	if len(dailyHours) > 0 {
		mean := total / float64(len(dailyHours))
		for _, h := range dailyHours {
			diff := h - mean
			variance += diff * diff
		}
		variance /= float64(len(dailyHours))
	}
	avg := 0.0
	if activeDays > 0 {
		avg = total / activeDays
	}
	features = append(features, total, activeDays, variance, avg, dailyHours[5], maxDay)

	for day := 0; day < 6; day++ {
		hasLunch := 0.0
		lateCount := 0.0
		for slot := 0; slot < 24; slot++ {
			subj := week[day][slot].GetSubjectID()
			if slot >= 8 && slot <= 11 && subj == 0 {
				hasLunch = 1.0
			}
			if slot >= 20 && subj > 0 {
				lateCount++
			}
		}
		features = append(features, hasLunch, lateCount)
	}

	instructors := make(map[uint16]bool)
	rooms := make(map[uint16]bool)
	subjects := make(map[uint16]bool)
	instructorLoad := make(map[uint16]float64)
	for day := 0; day < 6; day++ {
		for slot := 0; slot < 24; slot++ {
			ts := week[day][slot]
			subj := ts.GetSubjectID()
			instr := ts.GetInstructorID()
			room := ts.GetRoomID()
			if subj > 0 {
				subjects[subj] = true
			}
			if instr > 0 {
				instructors[instr] = true
				instructorLoad[instr]++
			}
			if room > 0 {
				rooms[room] = true
			}
		}
	}
	loadVar, loadMax, loadMin, loadAvg := 0.0, 0.0, 0.0, 0.0
	if len(instructorLoad) > 0 {
		first := true
		for _, load := range instructorLoad {
			loadAvg += load
			if first || load > loadMax {
				loadMax = load
			}
			if first || load < loadMin {
				loadMin = load
			}
			first = false
		}
		loadAvg /= float64(len(instructorLoad))
		for _, load := range instructorLoad {
			diff := load - loadAvg
			loadVar += diff * diff
		}
		loadVar /= float64(len(instructorLoad))
	}
	features = append(features, float64(len(instructors)), float64(len(rooms)), float64(len(subjects)), loadVar, loadMax, loadMin, loadAvg)

	dailyGaps := make([]float64, 6)
	maxContinuous := 0.0
	for day := 0; day < 6; day++ {
		first := -1
		last := -1
		occupied := 0
		currentContinuous := 0
		for slot := 0; slot < 24; slot++ {
			if week[day][slot].GetSubjectID() > 0 {
				if first == -1 {
					first = slot
				}
				last = slot
				occupied++
				currentContinuous++
				if float64(currentContinuous) > maxContinuous {
					maxContinuous = float64(currentContinuous)
				}
			} else {
				currentContinuous = 0
			}
		}
		if occupied > 0 {
			dailyGaps[day] = float64((last - first + 1) - occupied)
		}
	}
	totalGaps := 0.0
	for _, gaps := range dailyGaps {
		totalGaps += gaps
	}
	features = append(features, dailyGaps...)
	features = append(features, totalGaps, totalGaps/6.0, maxContinuous)

	morning, afternoon, evening := 0.0, 0.0, 0.0
	firstOccupied := -1
	lastOccupied := -1
	occupiedTotal := 0
	for day := 0; day < 6; day++ {
		for slot := 0; slot < 24; slot++ {
			if week[day][slot].GetSubjectID() == 0 {
				continue
			}
			flat := day*24 + slot
			if firstOccupied == -1 {
				firstOccupied = flat
			}
			lastOccupied = flat
			occupiedTotal++
			if slot < 8 {
				morning++
			} else if slot < 20 {
				afternoon++
			} else {
				evening++
			}
		}
	}
	totalSlots := morning + afternoon + evening
	morningRatio, afternoonRatio, eveningRatio := 0.0, 0.0, 0.0
	if totalSlots > 0 {
		morningRatio = morning / totalSlots
		afternoonRatio = afternoon / totalSlots
		eveningRatio = evening / totalSlots
	}
	saturdayRatio := 0.0
	if total > 0 {
		saturdayRatio = dailyHours[5] / total
	}
	spread := 0.0
	if occupiedTotal > 1 {
		spread = float64(lastOccupied-firstOccupied) / float64(occupiedTotal)
	}
	features = append(features, morning, afternoon, evening, morningRatio, afternoonRatio, eveningRatio, saturdayRatio, spread)

	return features
}
