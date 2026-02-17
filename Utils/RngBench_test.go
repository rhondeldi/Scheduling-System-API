package Utils

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
)

func BenchmarkRngIntnDedicated(b *testing.B) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))
	for i := 0; i < b.N; i++ {
		rng.Intn(10_000_000)
	}
}

func BenchmarkRngIntnDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rand.Intn(10_000_000)
	}
}

func BenchmarkRngInt63nDedicated(b *testing.B) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))
	for i := 0; i < b.N; i++ {
		rng.Int63n(10_000_000)
	}
}

func BenchmarkRngInt63nDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rand.Int63n(10_000_000)
	}
}

func BenchmarkRngInt31nDedicated(b *testing.B) {
	// fastest in my machine
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))
	for i := 0; i < b.N; i++ {
		rng.Int31n(10_000_000)
	}
}

func BenchmarkRngInt31nDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rand.Int31n(10_000_000)
	}
}

// max number of curriculums
const C int = 250

// max number year level out of all curriculums / courses.
const Y int = 4

// max number of section per year level.
const X int = 7

// max number of subjects in a semester.
const S int = 10

// total time slot in a week
const T int = Const.N_WEEKLY_TIME_SLOTS

// max number of instructor per department
const I int = 40

// max number of a specific room type per department
const R = 40

// time complexity
const O int = C * Y * X * S * 2 * T * (I + R)

func BenchmarkEstimatedGenPopTimeComplexity(b *testing.B) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))

	fmt.Printf(
		"Time Complexity | O( C * Y * X * S * 2 * T * (I + R) : O(%d * %d * %d * %d * 2 * %d * (%d + %d)) = %d\n",
		C, Y, X, S, T, I, R, O,
	)

	total := uint64(0)
	for i := 0; i < b.N; i++ {
		for gen_pop_op := 0; gen_pop_op < O; gen_pop_op++ {

			if (gen_pop_op % 1_000_000_000) == 0 {
				fmt.Printf("iter: %d, gen_pop_op: %d\n", i, gen_pop_op)
			}

			if (i % 3) == 0 {
				total += uint64(rng.Int31n(10_000_000))
			}
		}
	}
}
