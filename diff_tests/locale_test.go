package diff_tests

import (
	"testing"

	b0gus_assets "b0gus/assets"

	assert "github.com/stretchr/testify/assert"
)

func TestLocale(t *testing.T) {
	payload := b0gus_assets.GetLocalizedMsg("zh_CN", "main.DatabaseEmptyError", nil)
	assert.Equal(t, payload, "按给定配置获取到了空的数据库操作符。")
	payload = b0gus_assets.GetLocalizedMsg("english", "main.DatabaseChangingWarn", nil)
	assert.Equal(t, payload, "Won't update the database handler")
}
