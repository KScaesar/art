package Artifex

import (
	"reflect"
	"testing"
)

func TestAppData(t *testing.T) {
	type Label struct {
		ChatId string
		GameId string
	}

	adp, err := NewPublisherOption[string]().Build()
	if err != nil {
		t.Errorf("build apdater :unexpected error: got %v", err)
		return
	}

	key := "label"
	expectedCreatedData := &Label{ChatId: "xx123", GameId: "yy987"}
	expectedUpdateGameId := "zz987"
	filter := func(data *Label) bool {
		return data.ChatId == "xx123"
	}

	CreateAppData(adp, key, expectedCreatedData)
	gotData, found := GetAppData(adp, true, key, filter)
	if !found {
		t.Errorf("GetAppData() gotData not found")
		return
	}
	if !reflect.DeepEqual(gotData, expectedCreatedData) {
		t.Errorf("GetAppData() got=%v, want=%v", gotData, expectedCreatedData)
		return
	}

	UpdateAppData(adp, key, func(data *Label) { data.GameId = expectedUpdateGameId })
	gotData, _ = GetAppData(adp, true, key, filter)
	if gotData.GameId != expectedUpdateGameId {
		t.Errorf("UpdateAppData() got=%v, want=%v", gotData.GameId, expectedUpdateGameId)
		return
	}

	DeleteAppData(adp, key)
	found = HasAppData(adp, true, key, filter)
	if found {
		t.Errorf("DeleteAppData() fail")
		return
	}
}
