SEARCH_PATTERN:=

build_dev:
	go build -ldflags "-s -w" -o app && $(MAKE) frontend_local && ./app

build:
	go build -tags netgo -ldflags '-s -w' -o app

uml: project_uml.puml
	@echo UML already generated

project_uml.puml:
	@echo Generating UML...
	@echo using: https://github.com/jfeliu007/goplantuml
	rm project_uml.puml
	@goplantuml -recursive -show-aggregations -show-aliases -show-compositions -show-connection-labels -show-implementations -aggregate-private-members ./ > project_uml.puml

devr_local:
	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data

devr:
	@echo Download Test Data
	rm -rf scheduling-system-temporary-data
	gh release --repo github.com/mrdcvlsc/scheduling-system-temporary-data download --archive zip --clobber
	unzip scheduling-system-temporary-data-tmp-data-v*.zip -d ./ -x '*.py' '*.md' '*.git*' '*/makefile'
	mv scheduling-system-temporary-data-tmp-data-v*/ scheduling-system-temporary-data
	rm scheduling-system-temporary-data-tmp-data-v*.zip
	zip -r scheduling-system-temporary-data.zip scheduling-system-temporary-data

# old download temp data
# gh release --repo github.com/mrdcvlsc/scheduling-system-temporary-data download --pattern *.zip --clobber
# rm -rf scheduling-system-temporary-data
# mkdir scheduling-system-temporary-data
# unzip release.zip -d ./scheduling-system-temporary-data

frontend:
	@echo Download Frontend
	gh release --repo github.com/mrdcvlsc/scheduling-system-frontend download --pattern dist.zip --clobber
	rm -rf dist
	mkdir dist
	unzip dist.zip

frontend_local_release:
	rm -rf dist
	sed -i 's/const DEV = true/const DEV = false/' ../scheduling-system-frontend/src/js/basics.js
	cd ../scheduling-system-frontend && npm run build && cp -R dist ../scheduling-system-backend
	sed -i 's/const DEV = false/const DEV = true/' ../scheduling-system-frontend/src/js/basics.js

clean:
	go clean -testcache
	go clean -cache
	go clean -modcache

test:
	go clean -testcache
	echo "running generate schedule test encoding individual"

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run TestEstimateResourceAvailabilityFirstSem ./GeneticAlgorithm -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run TestEstimateResourceAvailabilitySecondSem ./GeneticAlgorithm -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run TestNewPopulationFirstSem ./GeneticAlgorithm -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run TestNewPopulationSecondSem ./GeneticAlgorithm -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run TestNewPopulation1stSemWithDepartmentSelection ./GeneticAlgorithm -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run TestNewPopulation2ndSemWithDepartmentSelection ./GeneticAlgorithm -timeout 0

	echo "running test resource methods"

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run Test ./Resources/Curriculum -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run Test ./Resources/Instructors -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run Test ./Resources/Rooms -timeout 0

	echo "running test schedule data structures"

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run Test ./Schedule -timeout 0

	echo "running test persistence methods"

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run Test ./StorageResources -timeout 0

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run Test ./StorageSchedule -timeout 0

	echo "running test others"

	rm -rf scheduling-system-temporary-data
	cd ../scheduling-system-temporary-data && python unify-subjects.py && python pack.py && mv release.zip ../scheduling-system-backend
	unzip release.zip -d ./scheduling-system-temporary-data
	go clean -testcache && go test -run Test ./Tests/schedule_datastructure_basic -timeout 0

	go clean -testcache && go test -run Test ./Utils -timeout 0

testv:
	go clean -testcache
	go test ./... -v -p 1 -timeout 0

testvs:
	go clean -testcache && go test -run TestNewPopulation ./GeneticAlgorithm -v -timeout 0

gh_i_test:
	@echo "running integration tests"

	$(MAKE) devr
	go clean -testcache && go test -run TestIntegrationEditCurriculumSectionV1 ./ -timeout 0

	$(MAKE) devr
	go clean -testcache && go test -run TestIntegrationEditCurriculumSectionV2 ./ -timeout 0

gh_test_local:
	@echo "running generate schedule test encoding individual"

	$(MAKE) devr_local
	go clean -testcache && go test -run TestEstimateResourceAvailabilityFirstSem ./GeneticAlgorithm -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run TestEstimateResourceAvailabilitySecondSem ./GeneticAlgorithm -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run TestNewPopulationFirstSem ./GeneticAlgorithm -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run TestNewPopulationSecondSem ./GeneticAlgorithm -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run TestNewPopulation1stSemWithDepartmentSelection ./GeneticAlgorithm -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run TestNewPopulation2ndSemWithDepartmentSelection ./GeneticAlgorithm -timeout 0

	@echo "running test resource methods"

	$(MAKE) devr_local
	go clean -testcache && go test -run Test ./Resources/Curriculum -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run Test ./Resources/Instructors -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run Test ./Resources/Rooms -timeout 0

	@echo "running test schedule data structures"

	$(MAKE) devr_local
	go clean -testcache && go test -run Test ./Schedule -timeout 0

	@echo "running test persistence methods"

	$(MAKE) devr_local
	go clean -testcache && go test -run Test ./StorageResources -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run Test ./StorageSchedule -timeout 0

	@echo "running test others"

	$(MAKE) devr_local
	go clean -testcache && go test -run Test ./Tests/schedule_datastructure_basic -timeout 0
	
	go clean -testcache && go test -run Test ./Utils -timeout 0

gh_i_test_local:
	@echo "running integration tests"

	$(MAKE) devr_local
	go clean -testcache && go test -run TestIntegrationEditCurriculumSectionV1 ./ -timeout 0

	$(MAKE) devr_local
	go clean -testcache && go test -run TestIntegrationEditCurriculumSectionV2 ./ -timeout 0

bench:
	# we need to escape the dollar sign for the command: go test -run=^$ -bench=. ./...
	go test -run=^$$ -bench=. ./... -benchmem

benchmark:
	$(MAKE) devr_local
	go clean -testcache
	make devr_local && go test -run=BenchmarkIntegrationTest -bench=BenchmarkIntegrationTest -timeout 0

todo:
	python todo.py

find:
	grep -nr $(SEARCH_PATTERN) ./

test_api:
	./app & sleep 1 && node Tests/schedule-serialization.js

update:
	go get -u ./

update_global:
	go get -u all