package payload_test

import (
	"testing"

	"github.com/menmos/menmos-go/payload"
)

func Test_ParseExpression(t *testing.T) {

	type testCase struct {
		name     string
		src      map[string]interface{}
		expected payload.Expression
		wantErr  bool
	}

	cases := []testCase{
		{"tag basic", map[string]interface{}{"tag": "bing"}, payload.NewExpression().AndTag("bing"), false},
		{"hasKey basic", map[string]interface{}{"key": "bing"}, payload.NewExpression().AndHasKey("bing"), false},
		{"keyValue basic", map[string]interface{}{"key": "bing", "value": "bong"}, payload.NewExpression().AndKeyValue("bing", "bong"), false},
		{"parent basic", map[string]interface{}{"parent": "asdf"}, payload.NewExpression().AndParent("asdf"), false},
		{
			"and with sized slice",
			map[string]interface{}{
				"and": [2]interface{}{
					map[string]interface{}{"tag": "bing"},
					map[string]interface{}{"key": "bong"},
				},
			},
			payload.NewExpression(),
			true,
		},
		{
			"and with unsized slice",
			map[string]interface{}{
				"and": []interface{}{
					map[string]interface{}{"tag": "bing"},
					map[string]interface{}{"key": "bong"},
				},
			},
			payload.NewExpression().AndTag("bing").AndHasKey("bong"),
			false,
		},
	}

	for _, tCase := range cases {
		t.Run(tCase.name, func(t *testing.T) {
			actual, err := payload.ParseExpression(tCase.src)
			if (err != nil) != tCase.wantErr {
				t.Errorf("expectedErr=%v, gotErr=%v", tCase.wantErr, err)
				return
			}

			if err == nil {
				if actual != tCase.expected {
					t.Errorf("expected expression=%v, got %v", tCase.expected, actual)
					return
				}
			}
		})
	}

}
