package values

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/reeflective/flags/types"
)

func TestCounter_Set(t *testing.T) {
	t.Parallel()
	var err error

	initial := 0
	counter := (*types.Counter)(&initial)

	require.Equal(t, 0, initial)
	require.Equal(t, "0", counter.String())
	require.Equal(t, 0, counter.Get())
	require.True(t, counter.IsBoolFlag())
	require.True(t, counter.IsCumulative())

	err = counter.Set("")
	require.NoError(t, err)
	require.Equal(t, 1, initial)
	require.Equal(t, "1", counter.String())

	err = counter.Set("10")
	require.NoError(t, err)
	require.Equal(t, 10, initial)
	require.Equal(t, "10", counter.String())

	err = counter.Set("-1")
	require.NoError(t, err)
	require.Equal(t, 11, initial)
	require.Equal(t, "11", counter.String())

	err = counter.Set("b")
	require.Error(t, err, "strconv.ParseInt: parsing \"b\": invalid syntax")
	require.Equal(t, 11, initial)
	require.Equal(t, "11", counter.String())
}

func TestBoolValue_IsBoolFlag(t *testing.T) {
	t.Parallel()
	b := &boolValue{}
	require.True(t, b.IsBoolFlag())
}

func TestValidateValue_IsBoolFlag(t *testing.T) {
	t.Parallel()
	boolV := true
	v := &validateValue{Value: newBoolValue(&boolV)}
	require.True(t, v.IsBoolFlag())

	v = &validateValue{Value: newStringValue(strP("stringValue"))}
	require.False(t, v.IsBoolFlag())
}

func TestValidateValue_IsCumulative(t *testing.T) {
	t.Parallel()
	v := &validateValue{Value: newStringValue(strP("stringValue"))}
	require.False(t, v.IsCumulative())

	v = &validateValue{Value: newStringSliceValue(&[]string{})}
	require.True(t, v.IsCumulative())
}

func TestValidateValue_String(t *testing.T) {
	t.Parallel()
	v := &validateValue{Value: newStringValue(strP("stringValue"))}
	require.Equal(t, "stringValue", v.String())

	v = &validateValue{Value: nil}
	require.Empty(t, v.String())
}

func TestValidateValue_Set(t *testing.T) {
	t.Parallel()
	sV := strP("stringValue")
	v := &validateValue{Value: newStringValue(sV)}
	require.NoError(t, v.Set("newVal"))
	require.Equal(t, "newVal", *sV)

	v.validateFunc = func(_ string) error {
		return nil
	}
	require.NoError(t, v.Set("newVal"))

	v.validateFunc = func(val string) error {
		return fmt.Errorf("invalid %s", val)
	}
	require.EqualError(t, v.Set("newVal"), "invalid newVal")
}

func strP(value string) *string {
	return &value
}
