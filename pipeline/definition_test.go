package pipeline

import (
	"testing"
)

func TestReadValidPipeline(t *testing.T) {
	// validyamlSpec in common_test.go
	_, err := GetDefinition([]byte(validyamlSpec))

	if err == nil {
		t.Log("Pipeline Parser able to read parse yaml file")
	} else {
		t.Fatalf("Pipeline Parser must be able to parse yaml")
	}

}

func TestReadInvalidPipeline(t *testing.T) {

	_, err := GetDefinition([]byte("---invalid yaml string"))

	if err != nil {
		t.Log("Pipeline Parser returns error on invalid yaml file")
	} else {
		t.Fatalf("Pipeline Parser must return error on invalid yaml")
	}

}

func TestReadEmptyPipeline(t *testing.T) {

	_, err := GetDefinition([]byte{})

	if err != nil {
		t.Log("Pipeline Parser returns error on empty yaml file")
	} else {
		t.Fatalf("Pipeline Parser must return error on empty yaml file")
	}
}
