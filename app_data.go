package Artifex

import (
	"github.com/gookit/goutil/maputil"
)

//

func CreateAppData[T any](adp IAdapter, appKey string, data T) {
	adp.Update(func(_ *string, appData maputil.Data) {
		err := appData.SetByPath(appKey, data)
		if err != nil {
			panic(err)
		}
	})
}

func HasAppData[T any](adp IAdapter, lock bool, appKey string, filter func(data T) bool) bool {
	_, found := GetAppData(adp, lock, appKey, filter)
	return found
}

func GetAppData[T any](adp IAdapter, lock bool, appKey string, filter func(data T) bool) (data T, found bool) {
	query := func(_ string, appData maputil.Data) {
		t, ok := appData.GetByPath(appKey)
		if !ok {
			return
		}

		d := t.(T)
		if filter(d) {
			data = d
			found = true
		}
	}

	if lock {
		adp.QueryRLock(query)
		return
	}
	adp.Query(query)
	return
}

func UpdateAppData[T any](adp IAdapter, appKey string, update func(data T)) {
	has := false
	adp.Query(func(_ string, appData maputil.Data) {
		if appData.Has(appKey) {
			has = true
			return
		}
	})
	if !has {
		return
	}

	adp.Update(func(_ *string, appData maputil.Data) {
		t, ok := appData.GetByPath(appKey)
		if !ok {
			return
		}

		data := t.(T)
		update(data)
	})
}

func DeleteAppData(adp IAdapter, appKey string) {
	adp.Update(func(_ *string, appData maputil.Data) {
		delete(appData, appKey)
	})
}
