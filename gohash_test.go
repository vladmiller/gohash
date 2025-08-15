package gohash_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vladmiller/gohash"
)

var _ hash.Hash = (*testHasher)(nil)

// testHasher partially implements [hash.Hash] interface and used to test the values
// that are streamed to hasher.
type testHasher struct {
	result []byte
}

func newTestHasher() *testHasher {
	return &testHasher{
		result: make([]byte, 0),
	}
}

// BlockSize implements hash.Hash.
func (t *testHasher) BlockSize() int {
	panic("unimplemented")
}

// Size implements hash.Hash.
func (t *testHasher) Size() int {
	panic("unimplemented")
}

// Reset implements hash.Hash.
func (t *testHasher) Reset() {
	panic("unimplemented")
}

// Sum implements hash.Hash.
func (t *testHasher) Sum(b []byte) []byte {
	return t.result
}

// Write implements hash.Hash.
func (t *testHasher) Write(p []byte) (int, error) {
	t.result = append(t.result, p...)
	return len(p), nil
}

// ------------------------------------------------------------------------------------

func TestFrom(t *testing.T) {
	// Snapshot of all hashes is stored to ensure consistency over time when new versions are released.
	snapshot := make([]string, len(testCases))
	snapshotExists := false

	if file, err := os.ReadFile("snapshot.txt"); err == nil {
		snapshotExists = true
		if err := json.Unmarshal(file, &snapshot); err != nil {
			require.NoError(t, err)
		}
	}

	// If snapshot exists, then compare the hash, otherwise,
	// record hash to a snapshot.
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			hash, err := gohash.From(tc, newTestHasher())
			require.NoError(t, err)

			if !snapshotExists {
				snapshot[i] = hex.EncodeToString(hash)
				return
			}

			assert.Equal(t, snapshot[i], hex.EncodeToString(hash))
		})
	}

	// Run the entire test a few more times to ensure that hashes are always the same
	for range 50 {
		for i, tc := range testCases {
			t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
				hash, err := gohash.From(tc, newTestHasher())
				require.NoError(t, err)
				assert.Equal(t, snapshot[i], hex.EncodeToString(hash))
			})
		}
	}

	// If snapshot did not exist before, then write it as json

	if !snapshotExists {
		b, err := json.MarshalIndent(snapshot, "", "  ")
		require.NoError(t, err)

		err = os.WriteFile("snapshot.txt", b, 0600)
		require.NoError(t, err)

		t.Error("Snapshot has been created, run the test again.")
	}
}

// TestContinuityOfMaps_Sample1 ensures that map input produces the same hash over 500 iterations.
func TestContinuityOfMaps_Sample1(t *testing.T) {
	// Given
	input := map[interface{}]interface{}{
		"foo":   "bar",
		"bar":   0,
		42:      "answer",
		3.14:    "pi",
		true:    "boolean",
		"apple": 1,
	}

	hash, _ := gohash.From(input, sha256.New())
	for i := range 500 {
		hash1, _ := gohash.From(input, sha256.New())
		require.Equal(t, hash, hash1, "Hash changed on iter %d, cannot sort a map champ?", i)
	}
}

// TestFrom_Pointers ensures that [gohash.From] can deference pointers and that the hash of the
// pointer is equal to the hash of the original value.
func TestFrom_Pointers(t *testing.T) {
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("pointer test %d", i), func(t *testing.T) {
			// Skip nil pointer test cases since &(*int(nil)) creates **int
			// which is a different type and should have a different hash
			if tc == nil {
				t.Skip("Skipping nil value")
			}
			rv := reflect.ValueOf(tc)
			if rv.Kind() == reflect.Ptr && rv.IsNil() {
				t.Skip("Skipping nil pointer")
			}

			expected, err := gohash.From(tc, sha256.New())
			require.NoError(t, err)

			actual, err := gohash.From(&tc, sha256.New())
			require.NoError(t, err)

			assert.Equal(t, expected, actual)
		})
	}
}

func TestFrom_EmptyMaps(t *testing.T) {
	h1, err := gohash.From(map[int]string{}, sha256.New())
	require.NoError(t, err)

	h2, err := gohash.From(map[string]string{}, sha256.New())
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2)
}

func TestFrom_EmptyPointers(t *testing.T) {
	var v1 *int
	h1, err := gohash.From(v1, sha256.New())
	require.NoError(t, err)

	var v2 *string
	h2, err := gohash.From(v2, sha256.New())
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2)
}

func BenchmarkFrom(b *testing.B) {
	b.ReportAllocs()

	hasher := sha256.New()
	for i, tc := range testCases {
		b.Run(fmt.Sprintf("test_%d", i), func(b *testing.B) {
			b.ResetTimer()

			for b.Loop() {
				_, err := gohash.From(tc, hasher)
				require.NoError(b, err)
				hasher.Reset()
			}
		})
	}
}

func p[T any](i T) *T {
	return &i
}

var nilInt *int
var testCases = []any{
	1, p(1), p(p(p(1))), 2.5, -4, -4324239493924,

	// Signed integers
	int(-1), int8(-8), int16(-16), int32(-32), int64(-64),

	// Unsigned integers
	uint(1), uint8(8), uint16(16), uint32(32), uint64(64),
	uintptr(0x1000),

	// Floating point
	float32(3.14), float64(2.718),

	// Complex
	complex64(1 + 2i), complex128(3 + 4i),

	// Boolean
	true, false,

	// String
	"hello", "",

	// Aliases
	byte(255), rune('A'),

	// Slice
	[]any{1, 2, 3, 4, p(5)},

	// Map
	map[string]any{
		"test": 1,
		"2":    3,
		"4":    true,
		"5":    []any{1, 2, 3, 4, 5},
	},

	map[string]any{
		"nested": map[string]interface{}{
			"array": []int{1, 2, 3},
			"ptr":   p("test"),
		},
	},

	// Struct
	struct{}{},

	struct {
		Key int
	}{
		Key: 42,
	},

	func() any {
		t, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
		return struct {
			ID           int
			InstrumentID int
			Status       string
			ExecutedAt   time.Time
			SettlementAt time.Time
			Duration     time.Duration
			TraderID     uuid.UUID
			Counterparty []struct {
				ID   int
				Firm string
			}
		}{
			ID:           247,
			InstrumentID: 89,
			Status:       "executed",
			ExecutedAt:   t,
			SettlementAt: t.Add(time.Hour * 72),
			Duration:     time.Duration((time.Hour * 72).Seconds()),
			TraderID:     uuid.MustParse("a1b2c3d4-5e6f-7890-abcd-ef1234567890"),
			Counterparty: []struct {
				ID   int
				Firm string
			}{
				{ID: 301, Firm: "Goldman Sachs"},
				{ID: 302, Firm: "JP Morgan"},
			},
		}
	}(),

	// Time
	func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:00Z")
		return t
	}(),

	time.Duration((time.Hour * 72).Seconds()),

	// examples from hashstructure
	nil,
	"foo",
	42,
	true,
	false,
	[]string{"foo", "bar"},
	[]interface{}{1, nil, "foo"},
	map[string]string{"foo": "bar"},
	map[interface{}]string{"foo": "bar"},
	map[interface{}]interface{}{"foo": "bar", "bar": 0},
	struct {
		Foo string
		Bar []interface{}
	}{
		Foo: "foo",
		Bar: []interface{}{nil, nil, nil},
	},
	&struct {
		Foo string
		Bar []interface{}
	}{
		Foo: "foo",
		Bar: []interface{}{nil, nil, nil},
	},

	// Pointer of pointer of pointer
	p(p(p(1))),

	(*int)(nil),
	nilInt,
}
