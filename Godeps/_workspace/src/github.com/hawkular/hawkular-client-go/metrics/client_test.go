package metrics

import (
	"crypto/rand"
	"fmt"
	assert "github.com/GoogleCloudPlatform/heapster/Godeps/_workspace/src/github.com/stretchr/testify/require"
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
	p := Parameters{Tenant: t, Host: "localhost:8080"}
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

func TestCreate(t *testing.T) {
	c, err := integrationClient()
	assert.Nil(t, err)

	id := "test.metric.create.numeric.1"
	md := MetricDefinition{Id: id, Type: Gauge}
	ok, err := c.Create(md)
	assert.Nil(t, err)
	assert.True(t, ok, "MetricDefinition should have been created")

	// Commented out, see HWKMETRICS-110
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
	assert.True(t, ok, "MetricDefinition should have been created")
	assert.Nil(t, err)

	// Fetch all the previously created metrics and test equalities..
	mdq, err := c.Definitions(Gauge)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(mdq), "Size of the returned gauge metrics does not match 2")

	mdm := make(map[string]MetricDefinition)
	for _, v := range mdq {
		mdm[v.Id] = *v
	}

	assert.Equal(t, md.Id, mdm[id].Id)
	assert.True(t, reflect.DeepEqual(tags, mdm["test.metric.create.numeric.2"].Tags))

	mda, err := c.Definitions(Availability)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(mda))
	assert.Equal(t, "test/metric/create/availability/1", mda[0].Id)
	assert.Equal(t, 12, mda[0].RetentionTime)

	if mda[0].Type != Availability {
		t.FailNow()
	}
}

func TestAddGaugeSingle(t *testing.T) {
	c, err := integrationClient()
	assert.Nil(t, err)

	// With timestamp
	m := Datapoint{Timestamp: time.Now().UnixNano() / 1e6, Value: 1.34}
	err = c.PushSingleGaugeMetric("test/numeric/single/1", m)
	assert.Nil(t, err)

	// Without preset timestamp
	m = Datapoint{Value: 2}
	err = c.PushSingleGaugeMetric("test.numeric.single.2", m)
	assert.Nil(t, err)

	//  for both metrics and check that they're correctly filled
	params := make(map[string]string)
	metrics, err := c.SingleGaugeMetric("test/numeric/single/1", params)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(metrics), "Received different amount of datapoints than sent")

	metrics, err = c.SingleGaugeMetric("test.numeric.single.2", params)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(metrics), "Received more datapoints than written")
	assert.False(t, metrics[0].Timestamp < 1, "Timestamp was not correctly populated")
}

func TestTagsModification(t *testing.T) {
	if c, err := integrationClient(); err == nil {
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

		assert.True(t, reflect.DeepEqual(tags, *md_tags), "Tags did not match the updated ones")

		// Delete some metric tags
		err = c.DeleteTags(Gauge, id, tags)
		assert.Nil(t, err)

		// Fetch metric - check that tags were deleted
		md_tags, err = c.Tags(Gauge, id)
		assert.Nil(t, err)
		assert.False(t, len(*md_tags) > 0, "Received deleted tags")
	}
}

func TestTags(t *testing.T) {
	if c, err := integrationClient(); err == nil {
		tags := make(map[string]string)
		tTag, err := randomString()
		tags[tTag] = "testValue"

		// Write with tags
		m := Datapoint{Value: float64(0.01), Tags: tags}
		err = c.PushSingleGaugeMetric("test.tags.numeric.1", m)
		assert.NoError(t, err)

		// Search metrics with tag

		// 		    @GET
		// @Path("/{tenantId}/numeric")
		// @ApiOperation(value = "Find numeric metrics data by their tags.", response = Map.cla@ApiParam(value = "Tag list", required = true) @Param("tags") Tags tags

		// Get metric definition tags
		// @Path("/{tenantId}/metrics/numeric/{id}/tags")
		// @ApiOperation(value = "Retrieve tags associated with the metric definition.", response = Metric.class)

		// Fetch a metric with values and check we still have tags
	}
}

func TestAddMixedMulti(t *testing.T) {

	// Modify to send both Availability as well as Gauge metrics at the same time
	if c, err := integrationClient(); err == nil {

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
			Type: Gauge,
		}

		h := []MetricHeader{hone, htwo}

		err = c.Write(h)
		assert.NoError(t, err)

		var checkDatapoints = func(id string, expected int) []*Datapoint {
			metric, err := c.SingleGaugeMetric(id, make(map[string]string))
			assert.NoError(t, err)
			assert.Equal(t, expected, len(metric), "Amount of datapoints does not match expected value")
			return metric
		}

		checkDatapoints("test.multi.numeric.1", 1)
		checkDatapoints("test.multi.numeric.2", 2)
	} else {
		t.Error(err)
	}
}

func TestCheckErrors(t *testing.T) {
	c, err := integrationClient()
	assert.Nil(t, err)

	err = c.PushSingleGaugeMetric("test.number.as.string", Datapoint{Value: "notFloat"})
	assert.NotNil(t, err, "Invalid non-float value should not be accepted")
	_, err = c.SingleGaugeMetric("test.not.existing", make(map[string]string))
	assert.Nil(t, err, "Querying empty metric should not generate an error")
}
