package cuirass_test

import (
	"testing"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/vaquita"
	"github.com/stretchr/testify/assert"
)

func TestGetPropertiesCached(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	p1 := cuirass.GetProperties(cfg, "n", "g")
	p2 := cuirass.GetProperties(cfg, "n", "g")

	assert.True(t, p1 == p2)
}

func TestGetPropertiesName(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	p1 := cuirass.GetProperties(cfg, "n", "g")
	p2 := cuirass.GetProperties(cfg, "n2", "g")

	assert.True(t, p1 != p2)
}

func TestGetPropertiesGroup(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	p1 := cuirass.GetProperties(cfg, "n", "g")
	p2 := cuirass.GetProperties(cfg, "n", "g2")

	assert.True(t, p1 != p2)
}

func TestGetPropertiesConfig(t *testing.T) {
	cfg := vaquita.NewEmptyMapConfig()
	p1 := cuirass.GetProperties(cfg, "n", "g")
	cfg2 := vaquita.NewEmptyMapConfig()
	p2 := cuirass.GetProperties(cfg2, "n", "g")

	assert.True(t, p1 != p2)
}
