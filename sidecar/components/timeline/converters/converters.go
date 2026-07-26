package converters

//go:generate go tool goverter gen .

import (
	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/components/timeline/models"
	enumscategories "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_categories"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

func zeroNilString(v string) *string {
	return utils.ZeroNil(v)
}

func parseTitleSource(v *string) *models.TitleSource {
	titleSource, err := models.ParseTitleSource(utils.Coalesce(v, ""))
	assert.NoError(err)

	return utils.ZeroNil(titleSource)
}

func parseBrowserCategory(v string) *enumscategories.Category {
	category, err := enumscategories.Parse(v)
	assert.NoError(err)

	return utils.ZeroNil(category)
}
