package Artifex

import (
	"github.com/gookit/goutil/maputil"
)

//

func CreateAppData[T any](adp IAdapter, appKey string, data T) {
	adp.Update(func(_ *string, appData maputil.Data) {
		appData.Set(appKey, data)
	})
}

func GetAppData[T any](adp IAdapter, lock bool, appKey string, filter func(data T) bool) (data T, found bool) {
	query := func(_ string, appData maputil.Data) {
		t, ok := appData.Get(appKey).(T)
		if ok && filter(t) {
			data = t
			found = true
			return
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
	adp.Update(func(_ *string, appData maputil.Data) {
		data, ok := appData.Get(appKey).(T)
		if ok {
			update(data)
		}
	})
}

func DeleteAppData[T any](adp IAdapter, appKey string) {
	adp.Update(func(_ *string, appData maputil.Data) {
		delete(appData, appKey)
	})
}
