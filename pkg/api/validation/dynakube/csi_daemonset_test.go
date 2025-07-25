package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
)

func TestDisabledCSIForReadonlyCSIVolume(t *testing.T) {
	objectMeta := defaultDynakubeObjectMeta.DeepCopy()
	objectMeta.Annotations = map[string]string{
		exp.CSIReadOnlyVolumeKey: "true",
	}

	t.Run("valid cloud-native dynakube specs", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: *objectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
		})
	})

	t.Run("invalid dynakube specs, as csi is not supported for feature", func(t *testing.T) {
		assertDenied(t,
			[]string{errorCSIEnabledRequired},
			&dynakube.DynaKube{
				ObjectMeta: *objectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testAPIURL,
					OneAgent: oneagent.Spec{
						ClassicFullStack: &oneagent.HostInjectSpec{},
					},
				},
			})
	})
}
