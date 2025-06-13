package gohash_test

// func TestJSONMarshalUnmarshal(t *testing.T) {
// 	// Given
// 	type testCase struct {
// 		A string
// 		B string
// 		C bool
// 		D int
// 	}

// 	var test testCase

// 	err := gofakeit.Struct(&test)
// 	assert.NoError(t, err)

// 	// Json struct
// 	type jsonStruct struct {
// 		Hash gohash.Hash
// 	}

// 	// Hash
// 	hash, err := gohash.From(test, sha256.New())
// 	assert.NoError(t, err)

// 	// Marshal
// 	j := jsonStruct{
// 		Hash: hash,
// 	}

// 	b, _ := json.Marshal(j)

// 	// Unmarshal
// 	var uj jsonStruct
// 	err = json.Unmarshal(b, &uj)
// 	assert.NoError(t, err)

// 	assert.Equal(t, j, uj)
// }
