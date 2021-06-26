// Copyright 2020 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package netdevice_test

import (
	"fmt"
	"testing"

	"github.com/jaypipes/ghw"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNetdevice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Netdevice Suite")
}

var _ = Describe("PciNetDevice", func() {
	Describe("creating new PciNetDevice", func() {
		t := GinkgoT()
		Context("succesfully", func() {
			It("should populate fields", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
					Files: map[string][]byte{
						"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0"),
						// real modalias data from a intel i350 PF - note trailing '\n'
						"sys/bus/pci/devices/0000:00:00.1/modalias": []byte("pci:v00008086d00001521sv0000FFFFsd00000000bc02sc00i00\n"),
					},
				}
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				pciDevs, err := ghw.PCI(ghw.WithChroot(fs.RootDir))
				Expect(err).ToNot(HaveOccurred())
				in := pciDevs.GetDevice("0000:00:00.1")
				Expect(in).ToNot(BeNil())

				f := factory.NewResourceFactory("fake", "fake", true)
				rc := &types.ResourceConfig{}

				dev, err := netdevice.NewPciNetDevice(in, f, rc)

				Expect(dev.GetDriver()).To(Equal("vfio-pci"))
				Expect(dev.GetNetName()).To(Equal("eth0"))
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(2)) // /dev/vfio/vfio0 and default /dev/vfio/vfio
				Expect(dev.GetRdmaSpec().IsRdma()).To(BeFalse())
				Expect(dev.GetRdmaSpec().GetRdmaDeviceSpec()).To(HaveLen(0))
				Expect(dev.GetLinkType()).To(Equal("fakeLinkType"))
				Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(dev.GetNumaInfo()).To(Equal("0"))
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not populate topology due to negative numa_node", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
					Files: map[string][]byte{
						"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("-1"),
						// real modalias data from a intel i350 PF - note trailing '\n'
						"sys/bus/pci/devices/0000:00:00.1/modalias": []byte("pci:v00008086d00001521sv0000FFFFsd00000000bc02sc00i00\n"),
					},
				}
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				pciDevs, err := ghw.PCI(ghw.WithChroot(fs.RootDir))
				Expect(err).ToNot(HaveOccurred())
				in := pciDevs.GetDevice("0000:00:00.1")
				Expect(in).ToNot(BeNil())

				f := factory.NewResourceFactory("fake", "fake", true)
				rc := &types.ResourceConfig{}

				dev, err := netdevice.NewPciNetDevice(in, f, rc)

				Expect(dev.GetAPIDevice().Topology).To(BeNil())
				Expect(dev.GetNumaInfo()).To(Equal(""))
				Expect(err).NotTo(HaveOccurred())
			})
			It("should not populate topology due to missing numa_node", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
					Files: map[string][]byte{
						// real modalias data from a intel i350 PF - note trailing '\n'
						"sys/bus/pci/devices/0000:00:00.1/modalias": []byte("pci:v00008086d00001521sv0000FFFFsd00000000bc02sc00i00\n"),
					},
				}
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				pciDevs, err := ghw.PCI(ghw.WithChroot(fs.RootDir))
				Expect(err).ToNot(HaveOccurred())
				in := pciDevs.GetDevice("0000:00:00.1")
				Expect(in).ToNot(BeNil())

				f := factory.NewResourceFactory("fake", "fake", true)
				rc := &types.ResourceConfig{}

				dev, err := netdevice.NewPciNetDevice(in, f, rc)

				Expect(dev.GetAPIDevice().Topology).To(BeNil())
				Expect(dev.GetNumaInfo()).To(Equal(""))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("with two devices but only one of them being RDMA", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObj: &types.NetDeviceSelectors{
					IsRdma: true,
				},
			}
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/devices/0000:00:00.2/net/eth1",
					"sys/bus/pci/drivers/mlx5_core",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/mlx5_core",
					"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/mlx5_core",
				},
				// real modalias data from a intel i350 PF - note trailing '\n'
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0"),
					"sys/bus/pci/devices/0000:00:00.1/modalias":  []byte("pci:v00008086d00001521sv0000FFFFsd00000000bc02sc00i00\n"),
					"sys/bus/pci/devices/0000:00:00.2/numa_node": []byte("0"),
					"sys/bus/pci/devices/0000:00:00.2/modalias":  []byte("pci:v00008086d00001521sv0000FFFFsd00000000bc02sc00i00\n"),
				},
			}
			defer fs.Use()()
			defer utils.UseFakeLinks()()

			rdma1 := &mocks.RdmaSpec{}
			// fake1 will have 2 RDMA device specs
			fake1ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			rdma1.On("IsRdma").Return(true).On("GetRdmaDeviceSpec").Return(fake1ds)

			// fake2 will have 0 rdma device specs to trigger error msg
			rdma2 := &mocks.RdmaSpec{}
			rdma2.On("IsRdma").Return(false)

			f := &mocks.ResourceFactory{}

			mockInfo1 := &mocks.DeviceInfoProvider{}
			mockInfo1.On("GetEnvVal").Return("0000:00:00.1")
			mockInfo1.On("GetDeviceSpecs").Return(nil)
			mockInfo1.On("GetMounts").Return(nil)
			mockInfo2 := &mocks.DeviceInfoProvider{}
			mockInfo2.On("GetEnvVal").Return("0000:00:00.2")
			mockInfo2.On("GetDeviceSpecs").Return(nil)
			mockInfo2.On("GetMounts").Return(nil)
			f.On("GetDefaultInfoProvider", "0000:00:00.1", "mlx5_core").Return(mockInfo1).
				On("GetDefaultInfoProvider", "0000:00:00.2", "mlx5_core").Return(mockInfo2).
				On("GetRdmaSpec", "0000:00:00.1").Return(rdma1).
				On("GetRdmaSpec", "0000:00:00.2").Return(rdma2)

			pciDevs, err := ghw.PCI(ghw.WithChroot(fs.RootDir))
			if err != nil {
				Fail(fmt.Sprintf("cannot load pci device info using ghw from %q: %v", fs.RootDir, err))
			}
			in1 := pciDevs.GetDevice("0000:00:00.1")
			in2 := pciDevs.GetDevice("0000:00:00.2")

			It("should populate Rdma device specs if isRdma", func() {
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				Expect(in1).ToNot(BeNil())

				dev, err := netdevice.NewPciNetDevice(in1, f, rc)

				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				Expect(dev.GetNetName()).To(Equal(""))
				Expect(dev.GetLinkType()).To(Equal(""))
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(2)) // 2x Rdma devs
				Expect(dev.GetMounts()).To(HaveLen(0))
				Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(dev.GetNumaInfo()).To(Equal("0"))
				Expect(err).NotTo(HaveOccurred())
				mockInfo1.AssertExpectations(t)
			})
			It("but not otherwise", func() {
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				Expect(in2).ToNot(BeNil())

				dev, err := netdevice.NewPciNetDevice(in2, f, rc)

				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				Expect(dev.GetNetName()).To(Equal("eth1"))
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.2"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev.GetMounts()).To(HaveLen(0))
				Expect(dev.GetLinkType()).To(Equal("fakeLinkType"))
				Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(dev.GetNumaInfo()).To(Equal("0"))
				Expect(err).NotTo(HaveOccurred())
				mockInfo2.AssertExpectations(t)
			})
		})
		Context("With needsVhostNet", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObj: &types.NetDeviceSelectors{
					NeedVhostNet: true,
				},
			}

			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/kernel/iommu_groups/0",
					"sys/bus/pci/drivers/vfio-pci",
					"dev/vhost-net",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
					"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0"),
					// real modalias data from a intel i350 PF - note trailing '\n'
					"sys/bus/pci/devices/0000:00:00.1/modalias": []byte("pci:v00008086d00001521sv0000FFFFsd00000000bc02sc00i00\n"),
				},
			}

			f := factory.NewResourceFactory("fake", "fake", true)
			It("should add the vhost-net deviceSpec", func() {
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				pciDevs, err := ghw.PCI(ghw.WithChroot(fs.RootDir))
				Expect(err).ToNot(HaveOccurred())
				in := pciDevs.GetDevice("0000:00:00.1")
				Expect(in).ToNot(BeNil())

				dev, err := netdevice.NewPciNetDevice(in, f, rc)

				Expect(dev.GetDriver()).To(Equal("vfio-pci"))
				Expect(dev.GetNetName()).To(Equal(""))
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(3)) // /dev/vfio/vfio0 and default /dev/vfio/vfio + vhost-net
				Expect(dev.GetRdmaSpec().IsRdma()).To(BeFalse())
				Expect(dev.GetRdmaSpec().GetRdmaDeviceSpec()).To(HaveLen(0))
				Expect(dev.GetLinkType()).To(Equal(""))
				Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(dev.GetNumaInfo()).To(Equal("0"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("cannot get device's driver", func() {
			It("should fail", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{"sys/bus/pci/devices/0000:00:00.1"},
					Files: map[string][]byte{
						"sys/bus/pci/devices/0000:00:00.1/driver": []byte("not a symlink"),
						// real modalias data from a intel i350 PF - note trailing '\n'
						"sys/bus/pci/devices/0000:00:00.1/modalias": []byte("pci:v00008086d00001521sv0000FFFFsd00000000bc02sc00i00\n"),
					},
				}
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				pciDevs, err := ghw.PCI(ghw.WithChroot(fs.RootDir))
				Expect(err).ToNot(HaveOccurred())

				in := pciDevs.GetDevice("0000:00:00.1")
				Expect(in).ToNot(BeNil())

				f := factory.NewResourceFactory("fake", "fake", true)
				rc := &types.ResourceConfig{}

				dev, err := netdevice.NewPciNetDevice(in, f, rc)
				Expect(dev).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
		Context("device's PF name is not available", func() {
			It("device should be added", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{"sys/bus/pci/devices/0000:00:00.1"},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
				}
				defer fs.Use()()
				defer utils.UseFakeLinks()()

				f := factory.NewResourceFactory("fake", "fake", true)
				in := &ghw.PCIDevice{
					Address: "0000:00:00.1",
					Driver:  "vfio-pci",
				}
				rc := &types.ResourceConfig{}

				dev, err := netdevice.NewPciNetDevice(in, f, rc)

				Expect(dev).NotTo(BeNil())
				Expect(dev.GetEnvVal()).To(Equal("0000:00:00.1"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
