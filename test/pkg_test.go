package tests

import (
	"context"
	"github.com/SENERGY-Platform/smart-service-module-worker-info/pkg"
	"github.com/SENERGY-Platform/smart-service-module-worker-info/test/mocks"
	"github.com/SENERGY-Platform/smart-service-module-worker-lib/pkg/configuration"
	"os"
	"sync"
	"testing"
	"time"
)

const TEST_CASE_DIR = "./testcases/"

func TestWithMocks(t *testing.T) {
	libConf, err := configuration.LoadLibConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	conf, err := configuration.Load[pkg.Config]("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	libConf.CamundaWorkerWaitDurationInMs = 200

	infos, err := os.ReadDir(TEST_CASE_DIR)
	if err != nil {
		t.Error(err)
		return
	}
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() && isValidaForMockTest(TEST_CASE_DIR+name) {
			t.Run(name, func(t *testing.T) {
				runTest(t, TEST_CASE_DIR+name, conf, libConf)
			})
		}
	}
}

func isValidaForMockTest(dir string) bool {
	expectedFiles := []string{
		"camunda_tasks.json",
		"expected_smart_service_repo_requests.json",
	}
	infos, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	files := map[string]bool{}
	for _, info := range infos {
		if !info.IsDir() {
			files[info.Name()] = true
		}
	}
	for _, expected := range expectedFiles {
		if !files[expected] {
			return false
		}
	}
	return true
}

func runTest(t *testing.T, testCaseLocation string, config pkg.Config, libConf configuration.Config) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	camunda := mocks.NewCamundaMock()
	libConf.CamundaUrl = camunda.Start(ctx, wg)
	err := camunda.AddFileToQueue(testCaseLocation + "/camunda_tasks.json")
	if err != nil {
		t.Error(err)
		return
	}

	libConf.AuthEndpoint = mocks.Keycloak(ctx, wg)

	moduleListResponse, _ := os.ReadFile(testCaseLocation + "/module_list_response.json")

	smartServiceRepo := mocks.NewSmartServiceRepoMock(libConf, config, moduleListResponse)
	libConf.SmartServiceRepositoryUrl = smartServiceRepo.Start(ctx, wg)

	err = pkg.Start(ctx, wg, config, libConf)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)

	err = smartServiceRepo.CheckExpectedRequestsFromFileLocation(testCaseLocation + "/expected_smart_service_repo_requests.json")
	if err != nil {
		t.Error("/expected_smart_service_repo_requests.json", err)
	}
}
