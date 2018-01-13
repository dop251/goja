package goja

type mockObjectLikeStruct map[string]interface{}

func (mols mockObjectLikeStruct) GetObjectValue(key string) (val interface{}, exists bool) {
	if key == "self" {
		return mols, true
	}

	val, exists = mols[key]
	return
}

func (mols mockObjectLikeStruct) SetObjectValue(key string, val interface{}) {
	if key == "self" {
		return
	}

	mols[key] = val
}

func (mols mockObjectLikeStruct) GetObjectKeys() []string {
	keys := make([]string, 1, len(mols)+1)
	keys[0] = "self"
	for key := range mols {
		keys = append(keys, key)
	}

	return keys
}

func (mols mockObjectLikeStruct) GetObjectLength() int {
	return len(mols) + 1
}

func (mols mockObjectLikeStruct) DeleteObjectValue(key string) {
	delete(mols, key)
}
