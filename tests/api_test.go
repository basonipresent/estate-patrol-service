// This file will run automated tests for API.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const ApiUrl = "http://localhost:8080"

// unknownEstateID is a well-formed UUID that no estate will ever have.
const unknownEstateID = "00000000-0000-0000-0000-000000000000"

func TestApi(t *testing.T) {
	if testing.Short() {
		t.Skip("Skip API tests")
	}

	testcases := getTestCases()
	ctx := context.Background()
	client := &http.Client{}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			for idx := range tc.Steps {
				step := &tc.Steps[idx]
				request, err := step.Request(t, ctx, &tc)
				require.NoError(t, err)
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json")

				// Send request
				response, err := client.Do(request)

				require.NoError(t, err)
				defer response.Body.Close()

				// Check response
				ReadJsonResult(t, response, step)
				step.Expect(t, ctx, &tc, response, step.Result)
			}
		})
	}
}

func getTestCases() []TestCase {
	return []TestCase{
		{
			Name: "Test Error 1",
			Steps: []TestCaseStep{
				{
					Request: func(t *testing.T, ctx context.Context, tc *TestCase) (*http.Request, error) {
						return http.NewRequest("POST", ApiUrl+"/estate", nil)
					},
					Expect: ExpectBadRequest(),
				},
			},
		},
		{
			Name: "Test Error 2: Invalid Format",
			Steps: []TestCaseStep{
				{
					Request: SendRequestNewEstate(-1, -5),
					Expect:  ExpectBadRequest(),
				},
			},
		},
		{
			Name: "Test Error: Create Tree Out of Bound",
			Steps: []TestCaseStep{
				{
					Request: SendRequestNewEstate(10, 20),
					Expect:  ExpectNewEstateOk(),
				},
				{
					Request: SendRequestNewTree(5, 0, 0),
					Expect:  ExpectBadRequest(),
				},
			},
		},
		{
			Name: "Error: estate dimensions out of range",
			Steps: []TestCaseStep{
				{Request: SendRequestNewEstate(0, 1), Expect: ExpectBadRequest()},
				{Request: SendRequestNewEstate(50001, 1), Expect: ExpectBadRequest()},
				{Request: SendRequest("POST", "/estate", `{"length":10.5,"width":1}`), Expect: ExpectBadRequest()},
				{Request: SendRequest("POST", "/estate", `{"length":"10","width":1}`), Expect: ExpectBadRequest()},
				{Request: SendRequest("POST", "/estate", `{"length":`), Expect: ExpectBadRequest()},
			},
		},
		{
			Name: "Error: invalid trees",
			Steps: []TestCaseStep{
				{Request: SendRequestNewEstate(5, 1), Expect: ExpectNewEstateOk()},
				{Request: SendRequestNewTree(10, 6, 1), Expect: ExpectBadRequest()}, // x past length
				{Request: SendRequestNewTree(10, 1, 2), Expect: ExpectBadRequest()}, // y past width
				{Request: SendRequestNewTree(0, 1, 1), Expect: ExpectBadRequest()},
				{Request: SendRequestNewTree(31, 1, 1), Expect: ExpectBadRequest()},
				{Request: SendRequestNewTree(10, 2, 1), Expect: ExpectNewTreeOk()},
				{Request: SendRequestNewTree(20, 2, 1), Expect: ExpectBadRequest()}, // plot occupied
			},
		},
		{
			Name: "Error: unknown estate",
			Steps: []TestCaseStep{
				{Request: SendRequest("POST", "/estate/"+unknownEstateID+"/tree", `{"x":1,"y":1,"height":5}`), Expect: ExpectNotFound()},
				{Request: SendRequest("GET", "/estate/"+unknownEstateID+"/stats", ""), Expect: ExpectNotFound()},
				{Request: SendRequest("GET", "/estate/"+unknownEstateID+"/drone-plan", ""), Expect: ExpectNotFound()},
			},
		},
		{
			Name: "Error: malformed estate id",
			Steps: []TestCaseStep{
				{Request: SendRequest("POST", "/estate/not-a-uuid/tree", `{"x":1,"y":1,"height":5}`), Expect: ExpectBadRequest()},
				{Request: SendRequest("GET", "/estate/not-a-uuid/stats", ""), Expect: ExpectBadRequest()},
				{Request: SendRequest("GET", "/estate/not-a-uuid/drone-plan", ""), Expect: ExpectBadRequest()},
			},
		},
		{
			Name: "Error: invalid max_distance",
			Steps: []TestCaseStep{
				{Request: SendRequestNewEstate(5, 1), Expect: ExpectNewEstateOk()},
				{Request: SendRequest("GET", "/estate/{id}/drone-plan?max_distance=0", ""), Expect: ExpectBadRequest()},
				{Request: SendRequest("GET", "/estate/{id}/drone-plan?max_distance=-5", ""), Expect: ExpectBadRequest()},
				{Request: SendRequest("GET", "/estate/{id}/drone-plan?max_distance=abc", ""), Expect: ExpectBadRequest()},
			},
		},
		CreateNormalTestCase("Normal 1", []any{
			[]any{CreateEstate, 10, 20},
			[]any{CreateTree, 10, 5, 5},
			[]any{CreateTree, 20, 6, 5},
		}),
		CreateNormalTestCase("Normal 2", []any{
			[]any{CreateEstate, 5, 1},
			[]any{CreateTree, 10, 2, 1},
			[]any{CreateTree, 20, 3, 1},
			[]any{CreateTree, 10, 4, 1},
			[]any{GetStats, 3, 10, 20, 10},
			[]any{GetDronePlan, 0, 82},
		}),
		CreateNormalTestCase("Spec example", []any{
			[]any{CreateEstate, 5, 1},
			[]any{CreateTree, 5, 2, 1},
			[]any{CreateTree, 3, 3, 1},
			[]any{CreateTree, 4, 4, 1},
			[]any{GetStats, 3, 3, 5, 4},
			[]any{GetDronePlan, 0, 54},
		}),
		CreateNormalTestCase("Empty estate", []any{
			[]any{CreateEstate, 4, 3},
			[]any{GetStats, 0, 0, 0, 0},
			[]any{GetDronePlan, 0, 10*(4*3-1) + 2},
		}),
		CreateNormalTestCase("Even median and row turn", []any{
			[]any{CreateEstate, 3, 2},
			[]any{CreateTree, 3, 3, 1},
			[]any{CreateTree, 4, 3, 2}, // neighbour of (3,1) across the row turn
			[]any{GetStats, 2, 3, 4, 3.5},
			[]any{GetDronePlan, 0, 10*5 + 2 + 3 + 1 + 4},
		}),
		// Bonus: {GetDronePlanRest, max_distance, distance flown, rest x, rest y}.
		CreateNormalTestCase("Bonus: forced landing on the spec example", []any{
			[]any{CreateEstate, 5, 1},
			[]any{CreateTree, 5, 2, 1},
			[]any{CreateTree, 3, 3, 1},
			[]any{CreateTree, 4, 4, 1},
			[]any{GetDronePlanRest, 3, 3, 1, 1},     // mid climb
			[]any{GetDronePlanRest, 10, 10, 1, 1},   // mid move: plot it left
			[]any{GetDronePlanRest, 16, 16, 2, 1},   // exactly on arrival
			[]any{GetDronePlanRest, 27, 27, 3, 1},   // mid descent: plot it reached
			[]any{GetDronePlanRest, 1000, 54, 5, 1}, // battery outlasts the plan
		}),
		CreateNormalTestCase("Bonus: largest estate", []any{
			[]any{CreateEstate, 50000, 50000},
			[]any{GetDronePlanRest, 1 + 10*50000, 1 + 10*50000, 50000, 2},
		}),
		CreateNormalTestCase("Largest estate", []any{
			[]any{CreateEstate, 50000, 50000},
			[]any{CreateTree, 30, 1, 1},
			[]any{CreateTree, 1, 1, 50000},
			[]any{GetDronePlan, 0, 10*(50000*50000-1) + 2 + 30 + 30 + 1 + 1},
		}),
	}
}

type TestCase struct {
	Name  string
	Steps []TestCaseStep
}

type RequestFunc func(*testing.T, context.Context, *TestCase) (*http.Request, error)
type ExpectFunc func(*testing.T, context.Context, *TestCase, *http.Response, map[string]any)

type TestCaseStep struct {
	Request RequestFunc
	Expect  ExpectFunc
	Result  map[string]any
}

func ResponseContains(t *testing.T, resp *http.Response, text string) {
	body, err := io.ReadAll(resp.Body)
	bodyStr := string(body)
	require.NoError(t, err)
	require.Contains(t, bodyStr, text)
}

func ReadJsonResult(t *testing.T, resp *http.Response, step *TestCaseStep) {
	var result map[string]any
	err := json.NewDecoder(resp.Body).Decode(&result)
	step.Result = result
	require.NoError(t, err)
}

func RequireIsUUID(t *testing.T, value string) {
	_, err := uuid.Parse(value)
	require.NoError(t, err)
}

const (
	CreateEstate = iota
	CreateTree
	GetStats
	GetDronePlan
	GetDronePlanRest
)

func CreateNormalTestCase(name string, a []any) TestCase {
	tc := TestCase{}
	tc.Name = name

	for _, step := range a {
		switch step.([]any)[0].(int) {
		case CreateEstate:
			tc.Steps = append(tc.Steps, TestCaseStep{
				Request: SendRequestNewEstate(step.([]any)[1].(int), step.([]any)[2].(int)),
				Expect:  ExpectNewEstateOk(),
			})
		case CreateTree:
			tc.Steps = append(tc.Steps, TestCaseStep{
				Request: SendRequestNewTree(step.([]any)[1].(int), step.([]any)[2].(int), step.([]any)[3].(int)),
				Expect:  ExpectNewTreeOk(),
			})
		case GetStats:
			tc.Steps = append(tc.Steps, TestCaseStep{
				Request: SendRequestGetStats(),
				Expect:  ExpectGetStatsOk(step.([]any)[1].(int), step.([]any)[2].(int), step.([]any)[3].(int), toFloat(step.([]any)[4])),
			})
		case GetDronePlan:
			tc.Steps = append(tc.Steps, TestCaseStep{
				Request: SendRequestGetDronePlan(step.([]any)[1].(int)),
				Expect:  ExpectGetDronePlanOk(step.([]any)[2].(int)),
			})
		case GetDronePlanRest:
			tc.Steps = append(tc.Steps, TestCaseStep{
				Request: SendRequestGetDronePlan(step.([]any)[1].(int)),
				Expect:  ExpectGetDronePlanRest(step.([]any)[2].(int), step.([]any)[3].(int), step.([]any)[4].(int)),
			})
		}

	}
	return tc
}

// toFloat lets a median be written as 4 or 3.5 in the test tables.
func toFloat(v any) float64 {
	if i, ok := v.(int); ok {
		return float64(i)
	}
	return v.(float64)
}

// SendRequest sends a raw body to path. "{id}" in path is replaced by the
// estate created in the test case's first step.
func SendRequest(method, path, body string) RequestFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase) (*http.Request, error) {
		p := path
		if strings.Contains(p, "{id}") {
			p = strings.ReplaceAll(p, "{id}", tc.Steps[0].Result["id"].(string))
		}
		return http.NewRequest(method, ApiUrl+p, strings.NewReader(body))
	}
}

func SendRequestNewEstate(length, width int) RequestFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase) (*http.Request, error) {
		req := map[string]int{
			"length": length,
			"width":  width,
		}
		body, err := json.Marshal(req)
		require.NoError(t, err)
		return http.NewRequest("POST", ApiUrl+"/estate", bytes.NewReader(body))
	}
}

func ExpectNewEstateOk() ExpectFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase, resp *http.Response, data map[string]any) {
		RequireReturnIsUUID(t, resp, data)
	}
}

func SendRequestNewTree(height, x, y int) RequestFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase) (*http.Request, error) {
		req := map[string]int{
			"height": height,
			"x":      x,
			"y":      y,
		}
		id := tc.Steps[0].Result["id"].(string)
		body, err := json.Marshal(req)
		require.NoError(t, err)
		return http.NewRequest("POST", ApiUrl+"/estate/"+id+"/tree", bytes.NewReader(body))
	}
}

func ExpectNewTreeOk() ExpectFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase, resp *http.Response, data map[string]any) {
		RequireReturnIsUUID(t, resp, data)
	}
}

func SendRequestGetStats() RequestFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase) (*http.Request, error) {
		id := tc.Steps[0].Result["id"].(string)
		return http.NewRequest("GET", ApiUrl+"/estate/"+id+"/stats", nil)
	}
}

func ExpectGetStatsOk(count, min, max int, median float64) ExpectFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase, resp *http.Response, data map[string]any) {
		RequireStats(t, resp, data, count, min, max, median)
	}
}

func SendRequestGetDronePlan(distance int) RequestFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase) (*http.Request, error) {
		id := tc.Steps[0].Result["id"].(string)
		var url string

		if distance == 0 {
			url = fmt.Sprintf("%s/estate/%s/drone-plan", ApiUrl, id)
		} else {
			url = fmt.Sprintf("%s/estate/%s/drone-plan?max_distance=%d", ApiUrl, id, distance)
		}
		return http.NewRequest("GET", url, nil)
	}
}

func ExpectGetDronePlanOk(distance int) ExpectFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase, resp *http.Response, data map[string]any) {
		RequireDistance(t, resp, data, distance)
		require.NotContains(t, data, "rest")
	}
}

func ExpectGetDronePlanRest(distance, x, y int) ExpectFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase, resp *http.Response, data map[string]any) {
		RequireDistance(t, resp, data, distance)
		rest := data["rest"].(map[string]any)
		require.Equal(t, x, int(rest["x"].(float64)))
		require.Equal(t, y, int(rest["y"].(float64)))
	}
}

func RequireReturnIsUUID(t *testing.T, resp *http.Response, data map[string]any) {
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	RequireIsUUID(t, data["id"].(string))
}

func RequireStats(t *testing.T, resp *http.Response, data map[string]any, count, min, max int, median float64) {
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, count, int(data["count"].(float64)))
	require.Equal(t, min, int(data["min"].(float64)))
	require.Equal(t, max, int(data["max"].(float64)))
	require.Equal(t, median, data["median"].(float64))
}

func RequireDistance(t *testing.T, resp *http.Response, data map[string]any, distance int) {
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, distance, int(data["distance"].(float64)))
}

func ExpectBadRequest() ExpectFunc {
	return ExpectError(http.StatusBadRequest)
}

func ExpectNotFound() ExpectFunc {
	return ExpectError(http.StatusNotFound)
}

// ExpectError checks the status and that the body is the {"message": ...} error shape.
func ExpectError(status int) ExpectFunc {
	return func(t *testing.T, ctx context.Context, tc *TestCase, resp *http.Response, data map[string]any) {
		require.Equal(t, status, resp.StatusCode)
		require.NotEmpty(t, data["message"])
	}
}
