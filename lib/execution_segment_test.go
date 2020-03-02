/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2019 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package lib

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stringToES(t *testing.T, str string) *ExecutionSegment {
	es := new(ExecutionSegment)
	require.NoError(t, es.UnmarshalText([]byte(str)))
	return es
}

func TestExecutionSegmentEquals(t *testing.T) {
	t.Parallel()

	t.Run("nil segment to full", func(t *testing.T) {
		var nilEs *ExecutionSegment
		fullEs := stringToES(t, "0:1")
		require.True(t, nilEs.Equal(fullEs))
		require.True(t, fullEs.Equal(nilEs))
	})

	t.Run("To it's self", func(t *testing.T) {
		es := stringToES(t, "1/2:2/3")
		require.True(t, es.Equal(es))
	})
}

func TestExecutionSegmentNew(t *testing.T) {
	t.Parallel()
	t.Run("from is below zero", func(t *testing.T) {
		_, err := NewExecutionSegment(big.NewRat(-1, 1), big.NewRat(1, 1))
		require.Error(t, err)
	})
	t.Run("to is more than 1", func(t *testing.T) {
		_, err := NewExecutionSegment(big.NewRat(0, 1), big.NewRat(2, 1))
		require.Error(t, err)
	})
	t.Run("from is smaller than to", func(t *testing.T) {
		_, err := NewExecutionSegment(big.NewRat(1, 2), big.NewRat(1, 3))
		require.Error(t, err)
	})

	t.Run("from is equal to 'to'", func(t *testing.T) {
		_, err := NewExecutionSegment(big.NewRat(1, 2), big.NewRat(1, 2))
		require.Error(t, err)
	})
	t.Run("ok", func(t *testing.T) {
		_, err := NewExecutionSegment(big.NewRat(0, 1), big.NewRat(1, 1))
		require.NoError(t, err)
	})
}

func TestExecutionSegmentUnmarshalText(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		input  string
		output *ExecutionSegment
		isErr  bool
	}{
		{input: "0:1", output: &ExecutionSegment{from: zeroRat, to: oneRat}},
		{input: "0.5:0.75", output: &ExecutionSegment{from: big.NewRat(1, 2), to: big.NewRat(3, 4)}},
		{input: "1/2:3/4", output: &ExecutionSegment{from: big.NewRat(1, 2), to: big.NewRat(3, 4)}},
		{input: "50%:75%", output: &ExecutionSegment{from: big.NewRat(1, 2), to: big.NewRat(3, 4)}},
		{input: "2/4:75%", output: &ExecutionSegment{from: big.NewRat(1, 2), to: big.NewRat(3, 4)}},
		{input: "75%", output: &ExecutionSegment{from: zeroRat, to: big.NewRat(3, 4)}},
		{input: "125%", isErr: true},
		{input: "1a5%", isErr: true},
		{input: "1a5", isErr: true},
		{input: "1a5%:2/3", isErr: true},
		{input: "125%:250%", isErr: true},
		{input: "55%:50%", isErr: true},
		// TODO add more strange or not so strange cases
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.input, func(t *testing.T) {
			es := new(ExecutionSegment)
			err := es.UnmarshalText([]byte(testCase.input))
			if testCase.isErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.True(t, es.Equal(testCase.output))

			// see if unmarshalling a stringified segment gets you back the same segment
			err = es.UnmarshalText([]byte(es.String()))
			require.NoError(t, err)
			require.True(t, es.Equal(testCase.output))
		})
	}

	t.Run("Unmarshal nilSegment.String", func(t *testing.T) {
		var nilEs *ExecutionSegment
		nilEsStr := nilEs.String()
		require.Equal(t, "0:1", nilEsStr)

		es := new(ExecutionSegment)
		err := es.UnmarshalText([]byte(nilEsStr))
		require.NoError(t, err)
		require.True(t, es.Equal(nilEs))
	})
}

func TestExecutionSegmentSplit(t *testing.T) {
	t.Parallel()

	var nilEs *ExecutionSegment
	_, err := nilEs.Split(-1)
	require.Error(t, err)

	_, err = nilEs.Split(0)
	require.Error(t, err)

	segments, err := nilEs.Split(1)
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, "0:1", segments[0].String())

	segments, err = nilEs.Split(2)
	require.NoError(t, err)
	require.Len(t, segments, 2)
	assert.Equal(t, "0:1/2", segments[0].String())
	assert.Equal(t, "1/2:1", segments[1].String())

	segments, err = nilEs.Split(3)
	require.NoError(t, err)
	require.Len(t, segments, 3)
	assert.Equal(t, "0:1/3", segments[0].String())
	assert.Equal(t, "1/3:2/3", segments[1].String())
	assert.Equal(t, "2/3:1", segments[2].String())

	secondQuarter, err := NewExecutionSegment(big.NewRat(1, 4), big.NewRat(2, 4))
	require.NoError(t, err)

	segments, err = secondQuarter.Split(1)
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, "1/4:1/2", segments[0].String())

	segments, err = secondQuarter.Split(2)
	require.NoError(t, err)
	require.Len(t, segments, 2)
	assert.Equal(t, "1/4:3/8", segments[0].String())
	assert.Equal(t, "3/8:1/2", segments[1].String())

	segments, err = secondQuarter.Split(3)
	require.NoError(t, err)
	require.Len(t, segments, 3)
	assert.Equal(t, "1/4:1/3", segments[0].String())
	assert.Equal(t, "1/3:5/12", segments[1].String())
	assert.Equal(t, "5/12:1/2", segments[2].String())

	segments, err = secondQuarter.Split(4)
	require.NoError(t, err)
	require.Len(t, segments, 4)
	assert.Equal(t, "1/4:5/16", segments[0].String())
	assert.Equal(t, "5/16:3/8", segments[1].String())
	assert.Equal(t, "3/8:7/16", segments[2].String())
	assert.Equal(t, "7/16:1/2", segments[3].String())
}

func TestExecutionSegmentScale(t *testing.T) {
	t.Parallel()
	es := new(ExecutionSegment)
	require.NoError(t, es.UnmarshalText([]byte("0.5")))
	require.Equal(t, int64(1), es.Scale(2))
	require.Equal(t, int64(1), es.Scale(3))
	require.Equal(t, int64(5), es.Scale(10))

	require.NoError(t, es.UnmarshalText([]byte("0.5:1.0")))
	require.Equal(t, int64(1), es.Scale(2))
	require.Equal(t, int64(2), es.Scale(3))
	require.Equal(t, int64(5), es.Scale(10))
}

func TestExecutionSegmentCopyScaleRat(t *testing.T) {
	t.Parallel()
	es := new(ExecutionSegment)
	twoRat := big.NewRat(2, 1)
	threeRat := big.NewRat(3, 1)
	require.NoError(t, es.UnmarshalText([]byte("0.5")))
	require.Equal(t, oneRat, es.CopyScaleRat(twoRat))
	require.Equal(t, big.NewRat(3, 2), es.CopyScaleRat(threeRat))

	require.NoError(t, es.UnmarshalText([]byte("0.5:1.0")))
	require.Equal(t, oneRat, es.CopyScaleRat(twoRat))
	require.Equal(t, big.NewRat(3, 2), es.CopyScaleRat(threeRat))

	var nilEs *ExecutionSegment
	require.Equal(t, twoRat, nilEs.CopyScaleRat(twoRat))
	require.Equal(t, threeRat, nilEs.CopyScaleRat(threeRat))
}

func TestExecutionSegmentInPlaceScaleRat(t *testing.T) {
	t.Parallel()
	es := new(ExecutionSegment)
	twoRat := big.NewRat(2, 1)
	threeRat := big.NewRat(3, 1)
	threeSecondsRat := big.NewRat(3, 2)
	require.NoError(t, es.UnmarshalText([]byte("0.5")))
	require.Equal(t, oneRat, es.InPlaceScaleRat(twoRat))
	require.Equal(t, oneRat, twoRat)
	require.Equal(t, threeSecondsRat, es.InPlaceScaleRat(threeRat))
	require.Equal(t, threeSecondsRat, threeRat)

	es = stringToES(t, "0.5:1.0")
	twoRat = big.NewRat(2, 1)
	threeRat = big.NewRat(3, 1)
	require.Equal(t, oneRat, es.InPlaceScaleRat(twoRat))
	require.Equal(t, oneRat, twoRat)
	require.Equal(t, threeSecondsRat, es.InPlaceScaleRat(threeRat))
	require.Equal(t, threeSecondsRat, threeRat)

	var nilEs *ExecutionSegment
	twoRat = big.NewRat(2, 1)
	threeRat = big.NewRat(3, 1)
	require.Equal(t, big.NewRat(2, 1), nilEs.InPlaceScaleRat(twoRat))
	require.Equal(t, big.NewRat(2, 1), twoRat)
	require.Equal(t, big.NewRat(3, 1), nilEs.InPlaceScaleRat(threeRat))
	require.Equal(t, big.NewRat(3, 1), threeRat)
}

func TestExecutionSegmentSubSegment(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		base, sub, result *ExecutionSegment
	}{
		// TODO add more strange or not so strange cases
		{
			name:   "nil base",
			base:   (*ExecutionSegment)(nil),
			sub:    stringToES(t, "0.2:0.3"),
			result: stringToES(t, "0.2:0.3"),
		},

		{
			name:   "nil sub",
			base:   stringToES(t, "0.2:0.3"),
			sub:    (*ExecutionSegment)(nil),
			result: stringToES(t, "0.2:0.3"),
		},
		{
			name:   "doc example",
			base:   stringToES(t, "1/2:1"),
			sub:    stringToES(t, "0:1/2"),
			result: stringToES(t, "1/2:3/4"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.result, testCase.base.SubSegment(testCase.sub))
		})
	}
}

func TestSplitBadSegment(t *testing.T) {
	t.Parallel()
	es := &ExecutionSegment{from: oneRat, to: zeroRat}
	_, err := es.Split(5)
	require.Error(t, err)
}

func TestSegmentExecutionFloatLength(t *testing.T) {
	t.Parallel()
	t.Run("nil has 1.0", func(t *testing.T) {
		var nilEs *ExecutionSegment
		require.Equal(t, 1.0, nilEs.FloatLength())
	})

	testCases := []struct {
		es       *ExecutionSegment
		expected float64
	}{
		// TODO add more strange or not so strange cases
		{
			es:       stringToES(t, "1/2:1"),
			expected: 0.5,
		},
		{
			es:       stringToES(t, "1/3:1"),
			expected: 0.66666,
		},

		{
			es:       stringToES(t, "0:1/2"),
			expected: 0.5,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.es.String(), func(t *testing.T) {
			require.InEpsilon(t, testCase.expected, testCase.es.FloatLength(), 0.001)
		})
	}
}

func TestExecutionSegmentSequences(t *testing.T) {
	t.Parallel()

	_, err := NewExecutionSegmentSequence(stringToES(t, "0:1/3"), stringToES(t, "1/2:1"))
	assert.Error(t, err)
}

func TestExecutionSegmentStringSequences(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		seq         string
		expSegments []string
		expError    bool
		canReverse  bool
		// TODO: checks for least common denominator and maybe striped partitioning
	}{
		{seq: "", expSegments: nil},
		{seq: "0.5", expError: true},
		{seq: "1,1", expError: true},
		{seq: "-0.5,1", expError: true},
		{seq: "1/2,1/2", expError: true},
		{seq: "1/2,1/3", expError: true},
		{seq: "0,1,1/2", expError: true},
		{seq: "0.5,1", expSegments: []string{"1/2:1"}},
		{seq: "1/2,1", expSegments: []string{"1/2:1"}, canReverse: true},
		{seq: "1/3,2/3", expSegments: []string{"1/3:2/3"}, canReverse: true},
		{seq: "0,1/3,2/3", expSegments: []string{"0:1/3", "1/3:2/3"}, canReverse: true},
		{seq: "0,1/3,2/3,1", expSegments: []string{"0:1/3", "1/3:2/3", "2/3:1"}, canReverse: true},
		{seq: "0.5,0.7", expSegments: []string{"1/2:7/10"}},
		{seq: "0.5,0.7,1", expSegments: []string{"1/2:7/10", "7/10:1"}},
		{seq: "0,1/13,2/13,1/3,1/2,3/4,1", expSegments: []string{
			"0:1/13", "1/13:2/13", "2/13:1/3", "1/3:1/2", "1/2:3/4", "3/4:1",
		}, canReverse: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.seq, func(t *testing.T) {
			result, err := NewExecutionSegmentSequenceFromString(tc.seq)
			if tc.expError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, len(tc.expSegments), len(result))
			for i, expStrSeg := range tc.expSegments {
				expSeg, errl := NewExecutionSegmentFromString(expStrSeg)
				require.NoError(t, errl)
				assert.Truef(t, expSeg.Equal(result[i]), "Segment %d (%s) should be equal to %s", i, result[i], expSeg)
			}
			if tc.canReverse {
				assert.Equal(t, result.String(), tc.seq)
			}
		})
	}
}

// Return a randomly distributed sequence of numSegments amount of
// execution segments whose length totals 1.
func genRandomExecutionSegmentSequence(numSegments int) (ExecutionSegmentSequence, error) {
	const denom int = 1000

	bounds := make(map[int]struct{}, numSegments-1)
	for i := 0; i < numSegments-1; i++ {
		b := rand.Intn(denom-1) + 1
		// Avoid duplicates
		if _, ok := bounds[b]; ok {
			numSegments++
			continue
		}
		bounds[b] = struct{}{}
	}

	nums := make([]int, 0, len(bounds)+2)
	for k := range bounds {
		nums = append(nums, k)
	}
	nums = append(nums, []int{0, denom}...)

	sort.Ints(nums)

	segments := make([]*ExecutionSegment, 0, len(bounds)+1)
	denom64 := int64(denom)
	for i := 0; i < len(nums)-1; i++ {
		from, to := big.NewRat(int64(nums[i]), denom64), big.NewRat(int64(nums[i+1]), denom64)
		segment, err := NewExecutionSegment(from, to)
		if err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}

	return NewExecutionSegmentSequence(segments...)
}

// Ensure that the sum of scaling all execution segments in
// the same sequence with scaling factor M results in M itself.
func TestExecutionSegmentScaleConsistency(t *testing.T) {
	t.Parallel()

	const numTests = 10
	for i := 0; i < numTests; i++ {
		scale := rand.Int31n(99) + 2
		seq, err := genRandomExecutionSegmentSequence(rand.Intn(9) + 2)
		require.NoError(t, err)

		t.Run(fmt.Sprintf("%d_%s", scale, seq), func(t *testing.T) {
			var total int64
			for _, segment := range seq {
				total += segment.Scale(int64(scale))
			}
			assert.Equal(t, int64(scale), total)
		})
	}
}

func TestExecutionSegmentScaleNoWobble(t *testing.T) {
	t.Parallel()

	requireSegmentScaleGreater := func(t *testing.T, es *ExecutionSegment) {
		var i, lastResult int64
		for i = 1; i < 1000; i++ {
			result := es.Scale(i)
			require.GreaterOrEqual(t, result, lastResult)
			lastResult = result
		}
	}

	// Baseline full segment test
	t.Run("0:1", func(t *testing.T) {
		es, err := NewExecutionSegment(zeroRat, oneRat)
		require.NoError(t, err)
		requireSegmentScaleGreater(t, es)
	})

	// Random segments
	const numTests = 10
	for i := 0; i < numTests; i++ {
		seq, err := genRandomExecutionSegmentSequence(rand.Intn(3) + 2)
		require.NoError(t, err)

		es := seq[rand.Intn(len(seq))]
		t.Run(es.String(), func(t *testing.T) {
			requireSegmentScaleGreater(t, es)
		})
	}
}

func BenchmarkExecutionSegmentScale(b *testing.B) {
	es, err := NewExecutionSegmentFromString("73/716:431/588")
	require.NoError(b, err)
	for n := 0; n < b.N; n++ {
		es.Scale(5)
	}
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}