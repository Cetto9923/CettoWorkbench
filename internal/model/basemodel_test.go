package model

import (
	"reflect"
	"testing"
)

func TestBaseModel_IDTagParse(t *testing.T) {
	field, ok := reflect.TypeOf(BaseModel{}).FieldByName("ID")
	if !ok {
		t.Fatal("BaseModel has no field named ID")
	}

	gormTag := field.Tag.Get("gorm")
	wantTag := "column:id;primaryKey;autoIncrement"
	if gormTag != wantTag {
		t.Fatalf("BaseModel.ID gorm tag = %q, want %q", gormTag, wantTag)
	}
}
