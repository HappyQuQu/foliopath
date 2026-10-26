package ocr

import (
	_ "embed"
	"encoding/json"
	"reflect"
	"testing"
)

//go:embed conformance.json
var conformanceFixtures []byte

func TestProducerConsumerSharedResultConformance(t *testing.T) {
	var cases []struct {
		Name  string `json:"name"`
		Lines []struct {
			Text        string     `json:"text"`
			Box         [8]float64 `json:"box"`
			Detection   float64    `json:"detection_score"`
			Recognition float64    `json:"recognition_score"`
		} `json:"lines"`
		Valid bool     `json:"valid"`
		Text  []string `json:"text"`
	}
	if err := json.Unmarshal(conformanceFixtures, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			lines := make([]Line, 0, len(tc.Lines))
			for _, value := range tc.Lines {
				lines = append(lines, Line{Text: value.Text, Box: value.Box, DetectionScore: value.Detection, RecognitionScore: value.Recognition})
			}
			result, err := Normalize(lines)
			if (err == nil) != tc.Valid {
				t.Fatal(result, err)
			}
			if tc.Valid {
				text := []string{}
				for _, v := range result.Lines {
					text = append(text, v.Text)
				}
				if !reflect.DeepEqual(text, tc.Text) {
					t.Fatal(text, tc.Text)
				}
			}
		})
	}
}
