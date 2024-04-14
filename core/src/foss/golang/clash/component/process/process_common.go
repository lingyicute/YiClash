//go:build !(android && cmfa)

package process

import "github.com/lingyicute/yiclashcore/constant"

func FindPackageName(metadata *constant.Metadata) (string, error) {
	return "", nil
}
