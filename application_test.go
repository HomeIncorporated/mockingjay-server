package main

import (
	"errors"
	"log"
	"net/http"
	"testing"

	"fmt"
	"github.com/quii/mockingjay-server/mockingjay"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
)

const someMonkeyConfigString = "Hello, world"

func TestCompatabilityWithWildcards(t *testing.T) {

	app := defaultApplication(log.New(ioutil.Discard, "", log.Ldate|log.Ltime), 1)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("Got request", r.URL)
		if r.URL.String() == "/hello" {
			fmt.Fprint(w, "world")
		} else {
			http.Error(w, "Nope", http.StatusNotFound)
		}
	}))

	defer svr.Close()

	notWildcardPath := "examples/issue40/1.yaml"
	err := app.CheckCompatibility(notWildcardPath, svr.URL)
	assert.NoError(t, err)

	wildCardPath := "examples/issue40/*.yaml"
	err = app.CheckCompatibility(wildCardPath, svr.URL)
	assert.Equal(t, ErrCDCFail, err)

}

func TestItFailsWhenTheConfigFileCantBeLoaded(t *testing.T) {
	app := testApplication()
	app.configLoader = failingIOUtil

	configPath := "mockingjay config path"
	_, err := app.CreateServer(configPath, "", false, nil)

	assert.NotNil(t, err)
	assert.Equal(t, err, errIOError)

	err = app.CheckCompatibility(configPath, "some url")
	assert.NotNil(t, err)
	assert.Equal(t, err, errIOError)
}

func TestItFailsWhenTheConfigIsInvalid(t *testing.T) {
	app := testApplication()
	app.mockingjayLoader = failingMockingjayLoader

	_, err := app.CreateServer("mockingjay config path", "", false, nil)

	assert.NotNil(t, err, "Didnt get an error when the mockingjay config failed to load")
	assert.Equal(t, err, errMJLoaderError)
}

func TestCompatFailsWhenConfigIsInvalid(t *testing.T) {
	app := testApplication()
	app.mockingjayLoader = failingMockingjayLoader

	err := app.CheckCompatibility("mockingjay config path", "some url")

	assert.NotNil(t, err, "Didnt get an error when the mockingjay config failed to load")
	assert.Equal(t, err, errMJLoaderError)
}

func TestItFailsWhenTheMonkeyConfigIsInvalid(t *testing.T) {
	app := testApplication()

	_, err := app.CreateServer("mockingjay config path", "monkey config path", false, nil)

	assert.NotNil(t, err, "Didnt get an error when the monkey config failed to load")
	assert.Equal(t, err, errMonkeyLoadError)
}

func TestItReturnsCDCErrorIfCompatabilityFails(t *testing.T) {
	app := new(application)
	app.configLoader = passingIOUtil
	app.mockingjayLoader = passingMockingjayLoader
	app.logger = log.New(ioutil.Discard, "", 0)

	app.compatabilityChecker = fakeCompatabilityChecker{passes: false}

	cdcError := app.CheckCompatibility("mj config path", "http://someurl")

	assert.NotNil(t, cdcError, "Didn't get an error when compatability fails")
	assert.Equal(t, ErrCDCFail, cdcError)
}

func testApplication() *application {
	app := new(application)
	app.configLoader = passingIOUtil
	app.mockingjayLoader = passingMockingjayLoader
	app.mockingjayServerMaker = mockingjay.NewServer
	app.monkeyServerMaker = failingMonkeyServerMaker
	app.logger = log.New(ioutil.Discard, "mocking-jay: ", log.Ldate|log.Ltime)
	return app
}

func testMockingJayConfig() []mockingjay.FakeEndpoint {

	yaml := `
---
 - name: Test endpoint
   request:
     uri: /hello
     method: GET
   response:
     code: 200
     body: 'hello, world'
`

	m, err := mockingjay.NewFakeEndpoints([]byte(yaml))

	if err != nil {
		log.Fatal(err)
	}

	return m
}

func passingIOUtil(path string) ([][]byte, []string, error) {
	monkeyConfigBytes := []byte(someMonkeyConfigString)
	return [][]byte{monkeyConfigBytes}, []string{"lol.yaml"}, nil
}

var errIOError = errors.New("Couldn't load err from FS")

func failingIOUtil(path string) ([][]byte, []string, error) {
	return nil, nil, errIOError
}

var errMJLoaderError = errors.New("Couldnt load mj file")

func failingMockingjayLoader([]byte) ([]mockingjay.FakeEndpoint, error) {
	return nil, errMJLoaderError
}

func passingMockingjayLoader([]byte) ([]mockingjay.FakeEndpoint, error) {
	return testMockingJayConfig(), nil
}

var errMonkeyLoadError = errors.New("Couldn't load monkey file")

func failingMonkeyServerMaker(http.Handler, string) (http.Handler, error) {
	return nil, errMonkeyLoadError
}

type fakeCompatabilityChecker struct {
	passes bool
}

func (f fakeCompatabilityChecker) CheckCompatibility(endpoints []mockingjay.FakeEndpoint, realURL string) bool {
	return f.passes
}
