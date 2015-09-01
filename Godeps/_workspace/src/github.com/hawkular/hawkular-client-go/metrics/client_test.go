package metrics

import (
	"crypto/rand"
	"fmt"
	assert "github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"time"
)

func integrationClient() (*Client, error) {
	t, err := randomString()
	if err != nil {
		return nil, err
	}
	// p := Parameters{Tenant: t, Host: "localhost:8080", Path: "hawkular/metrics"}
	// p := Parameters{Tenant: t, Host: "localhost:8180"}
	p := Parameters{Tenant: t, Host: "192.168.1.105:8080"}
	// p := Parameters{Tenant: t, Host: "209.132.178.218:18080"}
	return NewHawkularClient(p)
}

func randomString() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%X", b[:]), nil
}

func createError(err error) {
}

func TestTenantModifier(t *testing.T) {
	c, err := integrationClient()
	assert.Nil(t, err)

	ot, _ := randomString()

	// Create for another tenant
	id := "test.metric.create.numeric.tenant.1"
	md := MetricDefinition{Id: id, Type: Gauge}

	ok, err := c.Create(md, Tenant(ot))
	assert.Nil(t, err)
	assert.True(t, ok, "MetricDefinition should have been created")

	// Try to fetch from default tenant - should fail
	mds, err := c.Definitions(Filters(TypeFilter(Gauge)))
	assert.Nil(t, err)
	assert.Nil(t, mds)

	// Try to fetch from the given tenant - should succeed
	mds, err = c.Definitions(Filters(TypeFilter(Gauge)), Tenant(ot))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(mds))
}

func TestCreate(t *testing.T) {
	c, err := integrationClient()
	assert.Nil(t, err)

	id := "test.metric.create.numeric.1"
	md := MetricDefinition{Id: id, Type: Gauge}
	ok, err := c.Create(md)
	assert.Nil(t, err)
	assert.True(t, ok, "MetricDefinition should have been created")

	// Following would be nice:
	// mdd, err := c.Definitions(Filters(Type(Gauge), Id(id)))

	// mdd, err := c.Definition(Gauge, id)
	// assert.Nil(t, err)
	// assert.Equal(t, md.Id, mdd.Id)

	// Try to recreate the same..
	ok, err = c.Create(md)
	assert.False(t, ok, "Should have received false when recreating them same metric")
	assert.Nil(t, err)

	// Use tags and dataRetention
	tags := make(map[string]string)
	tags["units"] = "bytes"
	tags["env"] = "unittest"
	md_tags := MetricDefinition{Id: "test.metric.create.numeric.2", Tags: tags, Type: Gauge}

	ok, err = c.Create(md_tags)
	assert.True(t, ok, "MetricDefinition should have been created")
	assert.Nil(t, err)

	md_reten := MetricDefinition{Id: "test/metric/create/availability/1", RetentionTime: 12, Type: Availability}
	ok, err = c.Create(md_reten)
	assert.Nil(t, err)
	assert.True(t, ok, "MetricDefinition should have been created")

	// Fetch all the previously created metrics and test equalities..
	mdq, err := c.Definitions(Filters(TypeFilter(Gauge)))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(mdq), "Size of the returned gauge metrics does not match 2")

	mdm := make(map[string]MetricDefinition)
	for _, v := range mdq {
		mdm[v.Id] = *v
	}

	assert.Equal(t, md.Id, mdm[id].Id)
	assert.True(t, reflect.DeepEqual(tags, mdm["test.metric.create.numeric.2"].Tags))

	mda, err := c.Definitions(Filters(TypeFilter(Availability)))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(mda))
	assert.Equal(t, "test/metric/create/availability/1", mda[0].Id)
	assert.Equal(t, 12, mda[0].RetentionTime)

	if mda[0].Type != Availability {
		assert.FailNow(t, "Type did not match Availability", int(mda[0].Type))
	}
}

func TestTagsModification(t *testing.T) {
	c, err := integrationClient()
	assert.Nil(t, err)
	id := "test/tags/modify/1"
	// Create metric without tags
	md := MetricDefinition{Id: id, Type: Gauge}
	ok, err := c.Create(md)
	assert.Nil(t, err)
	assert.True(t, ok, "MetricDefinition should have been created")

	// Add tags
	tags := make(map[string]string)
	tags["ab"] = "ac"
	tags["host"] = "test"
	err = c.UpdateTags(Gauge, id, tags)
	assert.Nil(t, err)

	// Fetch metric tags - check for equality
	md_tags, err := c.Tags(Gauge, id)
	assert.Nil(t, err)

	assert.True(t, reflect.DeepEqual(tags, md_tags), "Tags did not match the updated ones")

	// Delete some metric tags
	err = c.DeleteTags(Gauge, id, tags)
	assert.Nil(t, err)

	// Fetch metric - check that tags were deleted
	md_tags, err = c.Tags(Gauge, id)
	assert.Nil(t, err)
	assert.False(t, len(md_tags) > 0, "Received deleted tags")
}

func TestAddMixedMulti(t *testing.T) {

	// Modify to send both Availability as well as Gauge metrics at the same time
	c, err := integrationClient()
	assert.NoError(t, err)

	mone := Datapoint{Value: 1.45, Timestamp: UnixMilli(time.Now())}
	hone := MetricHeader{
		Id:   "test.multi.numeric.1",
		Data: []Datapoint{mone},
		Type: Gauge,
	}

	mtwo_1 := Datapoint{Value: 2, Timestamp: UnixMilli(time.Now())}

	mtwo_2_t := UnixMilli(time.Now()) - 1e3

	mtwo_2 := Datapoint{Value: float64(4.56), Timestamp: mtwo_2_t}
	htwo := MetricHeader{
		Id:   "test.multi.numeric.2",
		Data: []Datapoint{mtwo_1, mtwo_2},
		Type: Counter,
	}

	h := []MetricHeader{hone, htwo}

	err = c.Write(h)
	assert.NoError(t, err)

	time.Sleep(1000 * time.Millisecond)

	var checkDatapoints = func(id string, typ MetricType, expected int) []*Datapoint {
		metric, err := c.ReadMetric(typ, id)
		assert.NoError(t, err)
		assert.Equal(t, expected, len(metric), "Amount of datapoints does not match expected value")
		return metric
	}

	checkDatapoints(hone.Id, hone.Type, 1)
	checkDatapoints(htwo.Id, htwo.Type, 2)
}

func TestCheckErrors(t *testing.T) {
	c, err := integrationClient()
	assert.Nil(t, err)

	mH := MetricHeader{
		Id:   "test.number.as.string",
		Data: []Datapoint{Datapoint{Value: "notFloat"}},
		Type: Gauge,
	}

	err = c.Write([]MetricHeader{mH})
	assert.NotNil(t, err, "Invalid non-float value should not be accepted")
	_, err = c.SingleGaugeMetric("test.not.existing", make(map[string]string))
	assert.Nil(t, err, "Querying empty metric should not generate an error")
}
