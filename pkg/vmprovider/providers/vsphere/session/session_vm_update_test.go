// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package session_test

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware/govmomi/object"
	vimTypes "github.com/vmware/govmomi/vim25/types"

	vmopv1alpha1 "github.com/vmware-tanzu/vm-operator/api/v1alpha1"

	"github.com/vmware-tanzu/vm-operator/pkg/conditions"
	"github.com/vmware-tanzu/vm-operator/pkg/lib"
	"github.com/vmware-tanzu/vm-operator/pkg/vmprovider/providers/vsphere/constants"
	"github.com/vmware-tanzu/vm-operator/pkg/vmprovider/providers/vsphere/session"
	"github.com/vmware-tanzu/vm-operator/pkg/vmprovider/providers/vsphere/virtualmachine"
)

var _ = Describe("Update ConfigSpec", func() {

	var config *vimTypes.VirtualMachineConfigInfo
	var configSpec *vimTypes.VirtualMachineConfigSpec

	BeforeEach(func() {
		config = &vimTypes.VirtualMachineConfigInfo{}
		configSpec = &vimTypes.VirtualMachineConfigSpec{}
	})

	// Just a few examples for testing these things here. Need to think more about whether this
	// is a good way or not. Probably better to do this via UpdateVirtualMachine when we have
	// better integration tests.

	Context("Basic Hardware", func() {
		var vmClassSpec *vmopv1alpha1.VirtualMachineClassSpec

		BeforeEach(func() {
			vmClassSpec = &vmopv1alpha1.VirtualMachineClassSpec{}
		})

		JustBeforeEach(func() {
			session.UpdateHardwareConfigSpec(config, configSpec, vmClassSpec)
		})

		It("config spec is empty", func() {
			Expect(configSpec.Annotation).ToNot(BeEmpty())
			Expect(configSpec.ManagedBy).ToNot(BeNil())
		})

		Context("Updates Hardware", func() {
			BeforeEach(func() {
				vmClassSpec.Hardware.Cpus = 42
				vmClassSpec.Hardware.Memory = resource.MustParse("2000Mi")
			})

			It("config spec is not empty", func() {
				Expect(configSpec.NumCPUs).To(BeNumerically("==", 42))
				Expect(configSpec.MemoryMB).To(BeNumerically("==", 2000))
			})
		})

		Context("config already matches", func() {
			BeforeEach(func() {
				config.Hardware.NumCPU = 42
				vmClassSpec.Hardware.Cpus = int64(config.Hardware.NumCPU)
				config.Hardware.MemoryMB = 1500
				vmClassSpec.Hardware.Memory = resource.MustParse(fmt.Sprintf("%dMi", config.Hardware.MemoryMB))
			})

			It("config spec show no changes", func() {
				Expect(configSpec.NumCPUs).To(BeZero())
				Expect(configSpec.MemoryMB).To(BeZero())
			})
		})
	})

	Context("CPU Allocation", func() {
		var vmClassSpec *vmopv1alpha1.VirtualMachineClassSpec
		var minCPUFreq uint64 = 1

		BeforeEach(func() {
			vmClassSpec = &vmopv1alpha1.VirtualMachineClassSpec{}
		})

		JustBeforeEach(func() {
			session.UpdateConfigSpecCPUAllocation(config, configSpec, vmClassSpec, minCPUFreq)
		})

		It("config spec is empty", func() {
			Expect(configSpec.CpuAllocation).To(BeNil())
		})

		Context("config matches class policy request", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.CpuAllocation = &vimTypes.ResourceAllocationInfo{
					Reservation: pointer.Int64Ptr(virtualmachine.CPUQuantityToMhz(r, minCPUFreq)),
				}
				vmClassSpec.Policies.Resources.Requests.Cpu = r
			})

			It("config spec is empty", func() {
				Expect(configSpec.CpuAllocation).To(BeNil())
			})
		})

		Context("config matches class policy limit", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.CpuAllocation = &vimTypes.ResourceAllocationInfo{
					Limit: pointer.Int64Ptr(virtualmachine.CPUQuantityToMhz(r, minCPUFreq)),
				}
				vmClassSpec.Policies.Resources.Limits.Cpu = r
			})

			It("config spec is empty", func() {
				Expect(configSpec.CpuAllocation).To(BeNil())
			})
		})

		Context("config matches is different from policy limit", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.CpuAllocation = &vimTypes.ResourceAllocationInfo{
					Limit: pointer.Int64Ptr(10 * virtualmachine.CPUQuantityToMhz(r, minCPUFreq)),
				}
				vmClassSpec.Policies.Resources.Limits.Cpu = r
			})

			It("config spec is not empty", func() {
				Expect(configSpec.CpuAllocation).ToNot(BeNil())
				Expect(configSpec.CpuAllocation.Reservation).To(BeNil())
				Expect(configSpec.CpuAllocation.Limit).ToNot(BeNil())
				Expect(*configSpec.CpuAllocation.Limit).To(BeNumerically("==", 100*1024*1024))
			})
		})

		Context("config matches is different from policy request", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.CpuAllocation = &vimTypes.ResourceAllocationInfo{
					Reservation: pointer.Int64Ptr(10 * virtualmachine.CPUQuantityToMhz(r, minCPUFreq)),
				}
				vmClassSpec.Policies.Resources.Requests.Cpu = r
			})

			It("config spec is not empty", func() {
				Expect(configSpec.CpuAllocation).ToNot(BeNil())
				Expect(configSpec.CpuAllocation.Limit).To(BeNil())
				Expect(configSpec.CpuAllocation.Reservation).ToNot(BeNil())
				Expect(*configSpec.CpuAllocation.Reservation).To(BeNumerically("==", 100*1024*1024))
			})
		})
	})

	Context("Memory Allocation", func() {
		var vmClassSpec *vmopv1alpha1.VirtualMachineClassSpec

		BeforeEach(func() {
			vmClassSpec = &vmopv1alpha1.VirtualMachineClassSpec{}
		})

		JustBeforeEach(func() {
			session.UpdateConfigSpecMemoryAllocation(config, configSpec, vmClassSpec)
		})

		It("config spec is empty", func() {
			Expect(configSpec.MemoryAllocation).To(BeNil())
		})

		Context("config matches class policy request", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.MemoryAllocation = &vimTypes.ResourceAllocationInfo{
					Reservation: pointer.Int64Ptr(virtualmachine.MemoryQuantityToMb(r)),
				}
				vmClassSpec.Policies.Resources.Requests.Memory = r
			})

			It("config spec is empty", func() {
				Expect(configSpec.MemoryAllocation).To(BeNil())
			})
		})

		Context("config matches class policy limit", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.MemoryAllocation = &vimTypes.ResourceAllocationInfo{
					Limit: pointer.Int64Ptr(virtualmachine.MemoryQuantityToMb(r)),
				}
				vmClassSpec.Policies.Resources.Limits.Memory = r
			})

			It("config spec is empty", func() {
				Expect(configSpec.MemoryAllocation).To(BeNil())
			})
		})

		Context("config matches is different from policy limit", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.MemoryAllocation = &vimTypes.ResourceAllocationInfo{
					Limit: pointer.Int64Ptr(10 * virtualmachine.MemoryQuantityToMb(r)),
				}
				vmClassSpec.Policies.Resources.Limits.Memory = r
			})

			It("config spec is not empty", func() {
				Expect(configSpec.MemoryAllocation).ToNot(BeNil())
				Expect(configSpec.MemoryAllocation.Reservation).To(BeNil())
				Expect(configSpec.MemoryAllocation.Limit).ToNot(BeNil())
				Expect(*configSpec.MemoryAllocation.Limit).To(BeNumerically("==", 100))
			})
		})

		Context("config matches is different from policy request", func() {
			BeforeEach(func() {
				r := resource.MustParse("100Mi")
				config.MemoryAllocation = &vimTypes.ResourceAllocationInfo{
					Reservation: pointer.Int64Ptr(10 * virtualmachine.MemoryQuantityToMb(r)),
				}
				vmClassSpec.Policies.Resources.Requests.Memory = r
			})

			It("config spec is not empty", func() {
				Expect(configSpec.MemoryAllocation).ToNot(BeNil())
				Expect(configSpec.MemoryAllocation.Limit).To(BeNil())
				Expect(configSpec.MemoryAllocation.Reservation).ToNot(BeNil())
				Expect(*configSpec.MemoryAllocation.Reservation).To(BeNumerically("==", 100))
			})
		})
	})

	Context("ExtraConfig", func() {
		var vmImage *vmopv1alpha1.VirtualMachineImage
		var vmClassSpec *vmopv1alpha1.VirtualMachineClassSpec
		var classConfigSpec *vimTypes.VirtualMachineConfigSpec
		var vm *vmopv1alpha1.VirtualMachine
		var globalExtraConfig map[string]string
		var ecMap map[string]string

		BeforeEach(func() {
			vmImage = &vmopv1alpha1.VirtualMachineImage{}
			vmClassSpec = &vmopv1alpha1.VirtualMachineClassSpec{}
			vm = &vmopv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
			}
			globalExtraConfig = make(map[string]string)
			classConfigSpec = nil
		})

		JustBeforeEach(func() {
			session.UpdateConfigSpecExtraConfig(
				config,
				configSpec,
				classConfigSpec,
				&vmImage.Status,
				vmClassSpec,
				vm,
				globalExtraConfig)

			ecMap = make(map[string]string)
			for _, ec := range configSpec.ExtraConfig {
				if optionValue := ec.GetOptionValue(); optionValue != nil {
					ecMap[optionValue.Key] = optionValue.Value.(string)
				}
			}
		})

		Context("Empty input", func() {
			It("No changes", func() {
				Expect(ecMap).To(BeEmpty())
			})
		})

		Context("Updates configSpec.ExtraConfig", func() {
			BeforeEach(func() {
				conditions.MarkTrue(vmImage, vmopv1alpha1.VirtualMachineImageV1Alpha1CompatibleCondition)
				config.ExtraConfig = append(config.ExtraConfig, &vimTypes.OptionValue{
					Key: constants.VMOperatorV1Alpha1ExtraConfigKey, Value: constants.VMOperatorV1Alpha1ConfigReady})
				globalExtraConfig["guestinfo.test"] = "test"
				globalExtraConfig["global"] = "test"
			})

			It("Expected configSpec.ExtraConfig", func() {
				By("VM Image compatible", func() {
					Expect(ecMap).To(HaveKeyWithValue("guestinfo.vmservice.defer-cloud-init", "enabled"))
				})

				By("Global map", func() {
					Expect(ecMap).To(HaveKeyWithValue("guestinfo.test", "test"))
					Expect(ecMap).To(HaveKeyWithValue("global", "test"))
				})
			})

			Context("When VM uses metadata transport types other than CloudInit", func() {
				BeforeEach(func() {
					vm.Spec.VmMetadata = &vmopv1alpha1.VirtualMachineMetadata{
						Transport:     vmopv1alpha1.VirtualMachineMetadataExtraConfigTransport,
						ConfigMapName: "dummy-config",
					}
				})
				It("defer cloud-init extra config is not enabled", func() {
					Expect(ecMap).To(HaveKeyWithValue("guestinfo.vmservice.defer-cloud-init", "enabled"))
				})
			})

			Context("When VM uses CloudInit metadata transport type", func() {
				BeforeEach(func() {
					vm.Spec.VmMetadata = &vmopv1alpha1.VirtualMachineMetadata{
						Transport:     vmopv1alpha1.VirtualMachineMetadataCloudInitTransport,
						ConfigMapName: "dummy-config",
					}
				})
				It("defer cloud-init extra config is not enabled", func() {
					Expect(ecMap).ToNot(HaveKeyWithValue("guestinfo.vmservice.defer-cloud-init", "enabled"))
				})
			})
		})

		Context("ExtraConfig value already exists", func() {
			BeforeEach(func() {
				config.ExtraConfig = append(config.ExtraConfig, &vimTypes.OptionValue{Key: "foo", Value: "bar"})
				globalExtraConfig["foo"] = "bar"
			})

			It("No changes", func() {
				Expect(ecMap).To(BeEmpty())
			})
		})

		Context("InstanceStorage related tests", func() {

			Context("When InstanceStorage is NOT configured on VM", func() {
				It("No Changes", func() {
					Expect(ecMap).To(BeEmpty())
				})
			})

			Context("When InstanceStorage is configured on VM", func() {
				BeforeEach(func() {
					vm.Spec.Volumes = append(vm.Spec.Volumes, vmopv1alpha1.VirtualMachineVolume{
						Name: "pvc-volume-1",
						PersistentVolumeClaim: &vmopv1alpha1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "pvc-volume-1",
							},
							InstanceVolumeClaim: &vmopv1alpha1.InstanceVolumeClaimVolumeSource{
								StorageClass: "dummyStorageClass",
								Size:         resource.MustParse("256Gi"),
							},
						},
					})
				})

				It("maintenance mode powerOff extraConfig should be added", func() {
					Expect(ecMap).To(HaveKeyWithValue(constants.MMPowerOffVMExtraConfigKey, constants.ExtraConfigTrue))
				})
			})
		})

		Context("ThunderPciDevices related test", func() {

			Context("when virtual devices are not present", func() {
				It("No Changes", func() {
					Expect(ecMap).To(BeEmpty())
				})
			})

			Context("when vGPU device is available", func() {
				BeforeEach(func() {
					vmClassSpec.Hardware.Devices = vmopv1alpha1.VirtualDevices{VGPUDevices: []vmopv1alpha1.VGPUDevice{
						{
							ProfileName: "test-vgpu-profile",
						},
					}}
				})

				It("maintenance mode powerOff extraConfig should be added", func() {
					Expect(ecMap).To(HaveKeyWithValue(constants.MMPowerOffVMExtraConfigKey, constants.ExtraConfigTrue))
				})

				It("PCI passthru MMIO extraConfig should be added", func() {
					Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOExtraConfigKey, constants.ExtraConfigTrue))
					Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOSizeExtraConfigKey, constants.PCIPassthruMMIOSizeDefault))
				})

				Context("when PCI passthru MMIO override annotation is set", func() {
					BeforeEach(func() {
						vm.Annotations[constants.PCIPassthruMMIOOverrideAnnotation] = "12345"
					})

					It("PCI passthru MMIO extraConfig should be set to override annotation value", func() {
						Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOExtraConfigKey, constants.ExtraConfigTrue))
						Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOSizeExtraConfigKey, "12345"))
					})
				})
			})

			Context("when DDPIO device is available", func() {
				BeforeEach(func() {
					vmClassSpec.Hardware.Devices = vmopv1alpha1.VirtualDevices{DynamicDirectPathIODevices: []vmopv1alpha1.DynamicDirectPathIODevice{
						{
							VendorID:    123,
							DeviceID:    24,
							CustomLabel: "",
						},
					}}
				})

				It("maintenance mode powerOff extraConfig should be added", func() {
					Expect(ecMap).To(HaveKeyWithValue(constants.MMPowerOffVMExtraConfigKey, constants.ExtraConfigTrue))
				})

				It("PCI passthru MMIO extraConfig should be added", func() {
					Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOExtraConfigKey, constants.ExtraConfigTrue))
					Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOSizeExtraConfigKey, constants.PCIPassthruMMIOSizeDefault))
				})

				Context("when PCI passthru MMIO override annotation is set", func() {
					BeforeEach(func() {
						vm.Annotations[constants.PCIPassthruMMIOOverrideAnnotation] = "12345"
					})

					It("PCI passthru MMIO extraConfig should be set to override annotation value", func() {
						Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOExtraConfigKey, constants.ExtraConfigTrue))
						Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOSizeExtraConfigKey, "12345"))
					})
				})
			})
		})

		Context("when VM_Class_as_Config_DaynDate FSS is enabled", func() {
			var oldVMClassAsConfigDaynDateFunc func() bool
			const dummyKey = "dummy-key"
			const dummyVal = "dummy-val"

			BeforeEach(func() {
				oldVMClassAsConfigDaynDateFunc = lib.IsVMClassAsConfigFSSDaynDateEnabled
				lib.IsVMClassAsConfigFSSDaynDateEnabled = func() bool {
					return true
				}
			})

			AfterEach(func() {
				lib.IsVMClassAsConfigFSSDaynDateEnabled = oldVMClassAsConfigDaynDateFunc
			})

			Context("classConfigSpec extra config is not nil", func() {
				BeforeEach(func() {
					classConfigSpec = &vimTypes.VirtualMachineConfigSpec{
						ExtraConfig: []vimTypes.BaseOptionValue{
							&vimTypes.OptionValue{
								Key:   dummyKey + "-1",
								Value: dummyVal + "-1",
							},
							&vimTypes.OptionValue{
								Key:   dummyKey + "-2",
								Value: dummyVal + "-2",
							},
						},
					}
					config.ExtraConfig = append(config.ExtraConfig, &vimTypes.OptionValue{Key: "hello", Value: "world"})
				})
				It("vm extra config overlaps with global extra config", func() {
					globalExtraConfig["hello"] = "world"

					Expect(ecMap).To(HaveKeyWithValue(dummyKey+"-1", dummyVal+"-1"))
					Expect(ecMap).To(HaveKeyWithValue(dummyKey+"-2", dummyVal+"-2"))
					Expect(ecMap).ToNot(HaveKeyWithValue("hello", "world"))
				})

				It("global extra config overlaps with class config spec - class config spec takes precedence", func() {
					globalExtraConfig[dummyKey+"-1"] = dummyVal + "-3"
					Expect(ecMap).To(HaveKeyWithValue(dummyKey+"-1", dummyVal+"-1"))
					Expect(ecMap).To(HaveKeyWithValue(dummyKey+"-2", dummyVal+"-2"))
				})

				Context("class config spec has vGPU and DDPIO devices", func() {
					BeforeEach(func() {
						classConfigSpec.DeviceChange = []vimTypes.BaseVirtualDeviceConfigSpec{
							&vimTypes.VirtualDeviceConfigSpec{
								Operation: vimTypes.VirtualDeviceConfigSpecOperationAdd,
								Device: &vimTypes.VirtualPCIPassthrough{
									VirtualDevice: vimTypes.VirtualDevice{
										Backing: &vimTypes.VirtualPCIPassthroughVmiopBackingInfo{
											Vgpu: "SampleProfile2",
										},
									},
								},
							},
							&vimTypes.VirtualDeviceConfigSpec{
								Operation: vimTypes.VirtualDeviceConfigSpecOperationAdd,
								Device: &vimTypes.VirtualPCIPassthrough{
									VirtualDevice: vimTypes.VirtualDevice{
										Backing: &vimTypes.VirtualPCIPassthroughDynamicBackingInfo{
											AllowedDevice: []vimTypes.VirtualPCIPassthroughAllowedDevice{
												{
													VendorId: 52,
													DeviceId: 53,
												},
											},
											CustomLabel: "SampleLabel2",
										},
									},
								},
							},
						}

					})

					It("extraConfig Map has MMIO and MMPowerOff related keys added", func() {
						Expect(ecMap).To(HaveKeyWithValue(constants.MMPowerOffVMExtraConfigKey, constants.ExtraConfigTrue))
						Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOExtraConfigKey, constants.ExtraConfigTrue))
						Expect(ecMap).To(HaveKeyWithValue(constants.PCIPassthruMMIOSizeExtraConfigKey, constants.PCIPassthruMMIOSizeDefault))
					})
				})
			})
		})
	})

	Context("ChangeBlockTracking", func() {
		var vmSpec vmopv1alpha1.VirtualMachineSpec
		var classConfigSpec *vimTypes.VirtualMachineConfigSpec

		BeforeEach(func() {
			vmSpec = vmopv1alpha1.VirtualMachineSpec{
				AdvancedOptions: &vmopv1alpha1.VirtualMachineAdvancedOptions{},
			}
			config.ChangeTrackingEnabled = nil
			classConfigSpec = nil
		})

		AfterEach(func() {
			configSpec.ChangeTrackingEnabled = nil
		})

		It("cbt and status cbt unset", func() {
			session.UpdateConfigSpecChangeBlockTracking(config, configSpec, classConfigSpec, vmSpec)
			Expect(configSpec.ChangeTrackingEnabled).To(BeNil())
		})

		It("configSpec cbt set to true", func() {
			config.ChangeTrackingEnabled = pointer.BoolPtr(true)
			vmSpec.AdvancedOptions.ChangeBlockTracking = pointer.BoolPtr(false)

			session.UpdateConfigSpecChangeBlockTracking(config, configSpec, classConfigSpec, vmSpec)
			Expect(configSpec.ChangeTrackingEnabled).ToNot(BeNil())
			Expect(*configSpec.ChangeTrackingEnabled).To(BeFalse())
		})

		It("configSpec cbt set to false", func() {
			config.ChangeTrackingEnabled = pointer.BoolPtr(false)
			vmSpec.AdvancedOptions.ChangeBlockTracking = pointer.BoolPtr(true)

			session.UpdateConfigSpecChangeBlockTracking(config, configSpec, classConfigSpec, vmSpec)
			Expect(configSpec.ChangeTrackingEnabled).ToNot(BeNil())
			Expect(*configSpec.ChangeTrackingEnabled).To(BeTrue())
		})

		It("configSpec cbt matches", func() {
			config.ChangeTrackingEnabled = pointer.BoolPtr(true)
			vmSpec.AdvancedOptions.ChangeBlockTracking = pointer.BoolPtr(true)

			session.UpdateConfigSpecChangeBlockTracking(config, configSpec, classConfigSpec, vmSpec)
			Expect(configSpec.ChangeTrackingEnabled).To(BeNil())
		})

		It("classConfigSpec not nil and is ignored", func() {
			config.ChangeTrackingEnabled = pointer.BoolPtr(false)
			vmSpec.AdvancedOptions.ChangeBlockTracking = pointer.BoolPtr(true)
			classConfigSpec = &vimTypes.VirtualMachineConfigSpec{
				ChangeTrackingEnabled: pointer.BoolPtr(false),
			}

			session.UpdateConfigSpecChangeBlockTracking(config, configSpec, classConfigSpec, vmSpec)
			Expect(configSpec.ChangeTrackingEnabled).ToNot(BeNil())
			Expect(*configSpec.ChangeTrackingEnabled).To(BeTrue())
		})

		Context("VM_Class_as_Config_DaynDate FSS is enabled", func() {
			var oldVMClassAsConfigDaynDateFunc func() bool
			BeforeEach(func() {
				oldVMClassAsConfigDaynDateFunc = lib.IsVMClassAsConfigFSSDaynDateEnabled
				lib.IsVMClassAsConfigFSSDaynDateEnabled = func() bool {
					return true
				}
				config.ChangeTrackingEnabled = pointer.BoolPtr(false)
				vmSpec.AdvancedOptions.ChangeBlockTracking = pointer.BoolPtr(true)
			})

			AfterEach(func() {
				lib.IsVMClassAsConfigFSSDaynDateEnabled = oldVMClassAsConfigDaynDateFunc
			})

			It("classConfigSpec not nil and same as configInfo", func() {
				classConfigSpec = &vimTypes.VirtualMachineConfigSpec{
					ChangeTrackingEnabled: pointer.BoolPtr(false),
				}

				session.UpdateConfigSpecChangeBlockTracking(config, configSpec, classConfigSpec, vmSpec)
				Expect(configSpec.ChangeTrackingEnabled).To(BeNil())
			})

			It("classConfigSpec not nil, different from configInfo, overrides vm spec cbt", func() {
				classConfigSpec = &vimTypes.VirtualMachineConfigSpec{
					ChangeTrackingEnabled: pointer.BoolPtr(true),
				}

				session.UpdateConfigSpecChangeBlockTracking(config, configSpec, classConfigSpec, vmSpec)
				Expect(configSpec.ChangeTrackingEnabled).ToNot(BeNil())
				Expect(*configSpec.ChangeTrackingEnabled).To(BeTrue())
			})
		})
	})

	Context("Firmware", func() {
		var vm *vmopv1alpha1.VirtualMachine

		BeforeEach(func() {
			vm = &vmopv1alpha1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
			}
			config.Firmware = "bios"
		})

		It("No firmware annotation", func() {
			session.UpdateConfigSpecFirmware(config, configSpec, vm)
			Expect(configSpec.Firmware).To(BeEmpty())
		})

		It("Set firmware annotation equal to current vm firmware", func() {
			vm.Annotations[constants.FirmwareOverrideAnnotation] = config.Firmware
			session.UpdateConfigSpecFirmware(config, configSpec, vm)
			Expect(configSpec.Firmware).To(BeEmpty())
		})

		It("Set firmware annotation differing to current vm firmware", func() {
			vm.Annotations[constants.FirmwareOverrideAnnotation] = "efi"
			session.UpdateConfigSpecFirmware(config, configSpec, vm)
			Expect(configSpec.Firmware).To(Equal("efi"))
		})

		It("Set firmware annotation to an invalid value", func() {
			vm.Annotations[constants.FirmwareOverrideAnnotation] = "invalidfirmware"
			session.UpdateConfigSpecFirmware(config, configSpec, vm)
			Expect(configSpec.Firmware).To(BeEmpty())
		})

	})

	Context("DeviceGroups", func() {
		var classConfigSpec *vimTypes.VirtualMachineConfigSpec

		BeforeEach(func() {
			classConfigSpec = &vimTypes.VirtualMachineConfigSpec{}
		})

		It("No DeviceGroups set in class config spec", func() {
			session.UpdateConfigSpecDeviceGroups(config, configSpec, classConfigSpec)
			Expect(configSpec.DeviceGroups).To(BeNil())
		})

		It("DeviceGroups set in class config spec", func() {
			classConfigSpec.DeviceGroups = &vimTypes.VirtualMachineVirtualDeviceGroups{
				DeviceGroup: []vimTypes.BaseVirtualMachineVirtualDeviceGroupsDeviceGroup{
					&vimTypes.VirtualMachineVirtualDeviceGroupsDeviceGroup{
						GroupInstanceKey: int32(400),
					},
				},
			}

			session.UpdateConfigSpecDeviceGroups(config, configSpec, classConfigSpec)
			Expect(configSpec.DeviceGroups).NotTo(BeNil())
			Expect(configSpec.DeviceGroups.DeviceGroup).To(HaveLen(1))
			deviceGroup := configSpec.DeviceGroups.DeviceGroup[0].GetVirtualMachineVirtualDeviceGroupsDeviceGroup()
			Expect(deviceGroup.GroupInstanceKey).To(Equal(int32(400)))
		})

		It("configInfo DeviceGroups set with vals different than the class config spec", func() {
			classConfigSpec.DeviceGroups = &vimTypes.VirtualMachineVirtualDeviceGroups{
				DeviceGroup: []vimTypes.BaseVirtualMachineVirtualDeviceGroupsDeviceGroup{
					&vimTypes.VirtualMachineVirtualDeviceGroupsDeviceGroup{
						GroupInstanceKey: int32(400),
					},
				},
			}

			config.DeviceGroups = &vimTypes.VirtualMachineVirtualDeviceGroups{
				DeviceGroup: []vimTypes.BaseVirtualMachineVirtualDeviceGroupsDeviceGroup{
					&vimTypes.VirtualMachineVirtualDeviceGroupsDeviceGroup{
						GroupInstanceKey: int32(500),
					},
				},
			}

			session.UpdateConfigSpecDeviceGroups(config, configSpec, classConfigSpec)
			Expect(configSpec.DeviceGroups).NotTo(BeNil())
			Expect(configSpec.DeviceGroups.DeviceGroup).To(HaveLen(1))
			deviceGroup := configSpec.DeviceGroups.DeviceGroup[0].GetVirtualMachineVirtualDeviceGroupsDeviceGroup()
			Expect(deviceGroup.GroupInstanceKey).To(Equal(int32(400)))
		})
	})

	Context("Ethernet Card Changes", func() {
		var expectedList object.VirtualDeviceList
		var currentList object.VirtualDeviceList
		var deviceChanges []vimTypes.BaseVirtualDeviceConfigSpec
		var dvpg1 *vimTypes.VirtualEthernetCardDistributedVirtualPortBackingInfo
		var dvpg2 *vimTypes.VirtualEthernetCardDistributedVirtualPortBackingInfo
		var err error

		BeforeEach(func() {
			dvpg1 = &vimTypes.VirtualEthernetCardDistributedVirtualPortBackingInfo{
				Port: vimTypes.DistributedVirtualSwitchPortConnection{
					PortgroupKey: "key1",
					SwitchUuid:   "uuid1",
				},
			}

			dvpg2 = &vimTypes.VirtualEthernetCardDistributedVirtualPortBackingInfo{
				Port: vimTypes.DistributedVirtualSwitchPortConnection{
					PortgroupKey: "key2",
					SwitchUuid:   "uuid2",
				},
			}
		})

		JustBeforeEach(func() {
			deviceChanges, err = session.UpdateEthCardDeviceChanges(expectedList, currentList)
		})

		AfterEach(func() {
			currentList = nil
			expectedList = nil
		})

		Context("No devices", func() {
			It("returns empty list", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(BeEmpty())
			})
		})

		Context("No device change when nothing changes", func() {
			var card1 vimTypes.BaseVirtualDevice
			var key1 int32 = 100
			var card2 vimTypes.BaseVirtualDevice
			var key2 int32 = 200

			BeforeEach(func() {
				card1, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card1.GetVirtualDevice().Key = key1
				expectedList = append(expectedList, card1)

				card2, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card2.GetVirtualDevice().Key = key2
				currentList = append(currentList, card2)
			})

			It("returns no device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(HaveLen(0))
			})
		})

		Context("Add device", func() {
			var card1 vimTypes.BaseVirtualDevice
			var key1 int32 = 100

			BeforeEach(func() {
				card1, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card1.GetVirtualDevice().Key = key1
				expectedList = append(expectedList, card1)
			})

			It("returns add device change", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(HaveLen(1))

				configSpec := deviceChanges[0].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card1.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
			})
		})

		Context("Add and remove device when backing change", func() {
			var card1 vimTypes.BaseVirtualDevice
			var card2 vimTypes.BaseVirtualDevice

			BeforeEach(func() {
				card1, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				expectedList = append(expectedList, card1)

				card2, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg2)
				Expect(err).ToNot(HaveOccurred())
				currentList = append(currentList, card2)
			})

			It("returns remove and add device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(HaveLen(2))

				configSpec := deviceChanges[0].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card2.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationRemove))

				configSpec = deviceChanges[1].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card1.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
			})
		})

		Context("Add and remove device when MAC address is different", func() {
			var card1 vimTypes.BaseVirtualDevice
			var key1 int32 = 100
			var card2 vimTypes.BaseVirtualDevice
			var key2 int32 = 200

			BeforeEach(func() {
				card1, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card1.GetVirtualDevice().Key = key1
				card1.(vimTypes.BaseVirtualEthernetCard).GetVirtualEthernetCard().AddressType = string(vimTypes.VirtualEthernetCardMacTypeManual)
				card1.(vimTypes.BaseVirtualEthernetCard).GetVirtualEthernetCard().MacAddress = "mac1"
				expectedList = append(expectedList, card1)

				card2, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card2.GetVirtualDevice().Key = key2
				card2.(vimTypes.BaseVirtualEthernetCard).GetVirtualEthernetCard().AddressType = string(vimTypes.VirtualEthernetCardMacTypeManual)
				card2.(vimTypes.BaseVirtualEthernetCard).GetVirtualEthernetCard().MacAddress = "mac2"
				currentList = append(currentList, card2)
			})

			It("returns remove and add device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(HaveLen(2))

				configSpec := deviceChanges[0].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card2.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationRemove))

				configSpec = deviceChanges[1].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card1.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
			})
		})

		Context("When WCP_VMClass_as_Config is enabled, Add and remove device when card type is different", func() {
			var card1 vimTypes.BaseVirtualDevice
			var key1 int32 = 100
			var card2 vimTypes.BaseVirtualDevice
			var key2 int32 = 200
			var oldVMClassAsConfigFunc func() bool

			BeforeEach(func() {
				oldVMClassAsConfigFunc = lib.IsVMClassAsConfigFSSEnabled
				lib.IsVMClassAsConfigFSSEnabled = func() bool {
					return true
				}
				card1, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card1.GetVirtualDevice().Key = key1
				expectedList = append(expectedList, card1)

				card2, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet2", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card2.GetVirtualDevice().Key = key2
				currentList = append(currentList, card2)
			})

			AfterEach(func() {
				lib.IsVMClassAsConfigFSSEnabled = oldVMClassAsConfigFunc
			})

			It("returns remove and add device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(HaveLen(2))

				configSpec := deviceChanges[0].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card2.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationRemove))

				configSpec = deviceChanges[1].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card1.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
			})
		})

		Context("Add and remove device when ExternalID is different", func() {
			var card1 vimTypes.BaseVirtualDevice
			var key1 int32 = 100
			var card2 vimTypes.BaseVirtualDevice
			var key2 int32 = 200

			BeforeEach(func() {
				card1, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card1.GetVirtualDevice().Key = key1
				card1.(vimTypes.BaseVirtualEthernetCard).GetVirtualEthernetCard().ExternalId = "ext1"
				expectedList = append(expectedList, card1)

				card2, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card2.GetVirtualDevice().Key = key2
				card2.(vimTypes.BaseVirtualEthernetCard).GetVirtualEthernetCard().ExternalId = "ext2"
				currentList = append(currentList, card2)
			})

			It("returns remove and add device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(HaveLen(2))

				configSpec := deviceChanges[0].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card2.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationRemove))

				configSpec = deviceChanges[1].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(card1.GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
			})
		})

		Context("Keeps existing device with same backing", func() {
			var card1 vimTypes.BaseVirtualDevice
			var key1 int32 = 100
			var card2 vimTypes.BaseVirtualDevice
			var key2 int32 = 200

			BeforeEach(func() {
				card1, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card1.GetVirtualDevice().Key = key1
				expectedList = append(expectedList, card1)

				card2, err = object.EthernetCardTypes().CreateEthernetCard("vmxnet3", dvpg1)
				Expect(err).ToNot(HaveOccurred())
				card2.GetVirtualDevice().Key = key2
				currentList = append(currentList, card2)
			})

			It("returns empty list", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(BeEmpty())
			})
		})
	})

	Context("Create vSphere PCI device", func() {
		var vgpuDevices = []vmopv1alpha1.VGPUDevice{
			{
				ProfileName: "SampleProfile",
			},
		}
		var ddpioDevices = []vmopv1alpha1.DynamicDirectPathIODevice{
			{
				VendorID:    42,
				DeviceID:    43,
				CustomLabel: "SampleLabel",
			},
		}
		var pciDevices vmopv1alpha1.VirtualDevices
		Context("For vGPU device", func() {
			BeforeEach(func() {
				pciDevices = vmopv1alpha1.VirtualDevices{
					VGPUDevices: vgpuDevices,
				}
			})
			It("should create vSphere device with VmiopBackingInfo", func() {
				vSphereDevices := virtualmachine.CreatePCIDevices(pciDevices, nil)
				Expect(vSphereDevices).To(HaveLen(1))
				virtualDevice := vSphereDevices[0].GetVirtualDevice()
				backing := virtualDevice.Backing.(*vimTypes.VirtualPCIPassthroughVmiopBackingInfo)
				Expect(backing.Vgpu).To(Equal(pciDevices.VGPUDevices[0].ProfileName))
			})
		})
		Context("For Dynamic DirectPath I/O device", func() {
			BeforeEach(func() {
				pciDevices = vmopv1alpha1.VirtualDevices{
					DynamicDirectPathIODevices: ddpioDevices,
				}
			})
			It("should create vSphere device with DynamicBackingInfo", func() {
				vSphereDevices := virtualmachine.CreatePCIDevices(pciDevices, nil)
				Expect(vSphereDevices).To(HaveLen(1))
				virtualDevice := vSphereDevices[0].GetVirtualDevice()
				backing := virtualDevice.Backing.(*vimTypes.VirtualPCIPassthroughDynamicBackingInfo)
				Expect(backing.AllowedDevice[0].DeviceId).To(Equal(int32(pciDevices.DynamicDirectPathIODevices[0].DeviceID)))
				Expect(backing.AllowedDevice[0].VendorId).To(Equal(int32(pciDevices.DynamicDirectPathIODevices[0].VendorID)))
				Expect(backing.CustomLabel).To(Equal(pciDevices.DynamicDirectPathIODevices[0].CustomLabel))
			})
		})
		Context("with VMClassAsConfig", func() {
			var (
				devIn []*vimTypes.VirtualPCIPassthrough
			)
			Context("For vGPU device", func() {
				BeforeEach(func() {
					pciDevices = vmopv1alpha1.VirtualDevices{
						VGPUDevices: []vmopv1alpha1.VGPUDevice{
							{
								ProfileName: "SampleProfile1",
							},
						},
					}
					devIn = []*vimTypes.VirtualPCIPassthrough{
						{
							VirtualDevice: vimTypes.VirtualDevice{
								Backing: &vimTypes.VirtualPCIPassthroughVmiopBackingInfo{
									Vgpu: "SampleProfile2",
								},
							},
						},
					}
				})
				It("should create two vGPUs", func() {
					devList := virtualmachine.CreatePCIDevices(pciDevices, devIn)
					Expect(devList).To(HaveLen(2))

					Expect(devList[0]).ToNot(BeNil())
					Expect(devList[0]).To(BeAssignableToTypeOf(&vimTypes.VirtualPCIPassthrough{}))
					Expect(devList[0].(*vimTypes.VirtualPCIPassthrough).Backing).ToNot(BeNil())
					Expect(devList[0].(*vimTypes.VirtualPCIPassthrough).Backing).To(BeAssignableToTypeOf(&vimTypes.VirtualPCIPassthroughVmiopBackingInfo{}))
					Expect(devList[0].(*vimTypes.VirtualPCIPassthrough).Backing.(*vimTypes.VirtualPCIPassthroughVmiopBackingInfo).Vgpu).To(Equal("SampleProfile1"))

					Expect(devList[1]).ToNot(BeNil())
					Expect(devList[1]).To(BeAssignableToTypeOf(&vimTypes.VirtualPCIPassthrough{}))
					Expect(devList[1].(*vimTypes.VirtualPCIPassthrough).Backing).ToNot(BeNil())
					Expect(devList[1].(*vimTypes.VirtualPCIPassthrough).Backing).To(BeAssignableToTypeOf(&vimTypes.VirtualPCIPassthroughVmiopBackingInfo{}))
					Expect(devList[1].(*vimTypes.VirtualPCIPassthrough).Backing.(*vimTypes.VirtualPCIPassthroughVmiopBackingInfo).Vgpu).To(Equal("SampleProfile2"))
				})
			})
		})
	})

	Context("PCI Device Changes", func() {
		var (
			currentList, expectedList object.VirtualDeviceList
			deviceChanges             []vimTypes.BaseVirtualDeviceConfigSpec
			err                       error

			// Variables related to vGPU devices.
			backingInfo1, backingInfo2 *vimTypes.VirtualPCIPassthroughVmiopBackingInfo
			deviceKey1, deviceKey2     int32
			vGPUDevice1, vGPUDevice2   vimTypes.BaseVirtualDevice

			// Variables related to dynamicDirectPathIO devices.
			allowedDev1, allowedDev2                         vimTypes.VirtualPCIPassthroughAllowedDevice
			backingInfo3, backingInfo4                       *vimTypes.VirtualPCIPassthroughDynamicBackingInfo
			deviceKey3, deviceKey4                           int32
			dynamicDirectPathIODev1, dynamicDirectPathIODev2 vimTypes.BaseVirtualDevice
		)

		BeforeEach(func() {
			backingInfo1 = &vimTypes.VirtualPCIPassthroughVmiopBackingInfo{Vgpu: "mockup-vmiop1"}
			backingInfo2 = &vimTypes.VirtualPCIPassthroughVmiopBackingInfo{Vgpu: "mockup-vmiop2"}
			deviceKey1 = int32(-200)
			deviceKey2 = int32(-201)
			vGPUDevice1 = virtualmachine.CreatePCIPassThroughDevice(deviceKey1, backingInfo1)
			vGPUDevice2 = virtualmachine.CreatePCIPassThroughDevice(deviceKey2, backingInfo2)

			allowedDev1 = vimTypes.VirtualPCIPassthroughAllowedDevice{
				VendorId: 1000,
				DeviceId: 100,
			}
			allowedDev2 = vimTypes.VirtualPCIPassthroughAllowedDevice{
				VendorId: 2000,
				DeviceId: 200,
			}
			backingInfo3 = &vimTypes.VirtualPCIPassthroughDynamicBackingInfo{
				AllowedDevice: []vimTypes.VirtualPCIPassthroughAllowedDevice{allowedDev1},
				CustomLabel:   "sampleLabel3",
			}
			backingInfo4 = &vimTypes.VirtualPCIPassthroughDynamicBackingInfo{
				AllowedDevice: []vimTypes.VirtualPCIPassthroughAllowedDevice{allowedDev2},
				CustomLabel:   "sampleLabel4",
			}
			deviceKey3 = int32(-202)
			deviceKey4 = int32(-203)
			dynamicDirectPathIODev1 = virtualmachine.CreatePCIPassThroughDevice(deviceKey3, backingInfo3)
			dynamicDirectPathIODev2 = virtualmachine.CreatePCIPassThroughDevice(deviceKey4, backingInfo4)
		})

		JustBeforeEach(func() {
			deviceChanges, err = session.UpdatePCIDeviceChanges(expectedList, currentList)
		})

		AfterEach(func() {
			currentList = nil
			expectedList = nil
		})

		Context("No devices", func() {
			It("returns empty list", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(BeEmpty())
			})
		})

		Context("Adding vGPU and dynamicDirectPathIO devices with different backing info", func() {
			BeforeEach(func() {
				expectedList = append(expectedList, vGPUDevice1)
				expectedList = append(expectedList, vGPUDevice2)
				expectedList = append(expectedList, dynamicDirectPathIODev1)
				expectedList = append(expectedList, dynamicDirectPathIODev2)
			})

			It("Should return add device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(len(deviceChanges)).To(Equal(len(expectedList)))

				for idx, dev := range deviceChanges {
					configSpec := dev.GetVirtualDeviceConfigSpec()
					Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(expectedList[idx].GetVirtualDevice().Key))
					Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
				}
			})
		})

		Context("Adding vGPU and dynamicDirectPathIO devices with same backing info", func() {
			BeforeEach(func() {
				expectedList = append(expectedList, vGPUDevice1)
				// Creating a vGPUDevice with same backingInfo1 but different deviceKey.
				vGPUDevice2 = virtualmachine.CreatePCIPassThroughDevice(deviceKey2, backingInfo1)
				expectedList = append(expectedList, vGPUDevice2)
				expectedList = append(expectedList, dynamicDirectPathIODev1)
				// Creating a dynamicDirectPathIO device with same backingInfo3 but different deviceKey.
				dynamicDirectPathIODev2 = virtualmachine.CreatePCIPassThroughDevice(deviceKey4, backingInfo3)
				expectedList = append(expectedList, dynamicDirectPathIODev2)
			})

			It("Should return add device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(len(deviceChanges)).To(Equal(len(expectedList)))

				for idx, dev := range deviceChanges {
					configSpec := dev.GetVirtualDeviceConfigSpec()
					Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(expectedList[idx].GetVirtualDevice().Key))
					Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
				}
			})
		})

		Context("When the expected and current lists have DDPIO devices with different custom labels", func() {
			BeforeEach(func() {
				expectedList = []vimTypes.BaseVirtualDevice{dynamicDirectPathIODev1}
				// Creating a dynamicDirectPathIO device with same backing info except for the custom label.
				backingInfoDiffCustomLabel := &vimTypes.VirtualPCIPassthroughDynamicBackingInfo{
					AllowedDevice: backingInfo3.AllowedDevice,
					CustomLabel:   "DifferentLabel",
				}
				dynamicDirectPathIODev2 = virtualmachine.CreatePCIPassThroughDevice(deviceKey4, backingInfoDiffCustomLabel)
				currentList = []vimTypes.BaseVirtualDevice{dynamicDirectPathIODev2}
			})

			It("should return add and remove device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(len(deviceChanges)).To(Equal(2))

				configSpec := deviceChanges[0].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(currentList[0].GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationRemove))

				configSpec = deviceChanges[1].GetVirtualDeviceConfigSpec()
				Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(expectedList[0].GetVirtualDevice().Key))
				Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
			})
		})

		Context("When the expected and current list of pciDevices have different Devices", func() {
			BeforeEach(func() {
				currentList = append(currentList, vGPUDevice1)
				expectedList = append(expectedList, vGPUDevice2)
				currentList = append(currentList, dynamicDirectPathIODev1)
				expectedList = append(expectedList, dynamicDirectPathIODev2)
			})

			It("Should return add and remove device changes", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(len(deviceChanges)).To(Equal(4))

				for i := 0; i < 2; i++ {
					configSpec := deviceChanges[i].GetVirtualDeviceConfigSpec()
					Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(currentList[i].GetVirtualDevice().Key))
					Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationRemove))
				}

				for i := 2; i < 4; i++ {
					configSpec := deviceChanges[i].GetVirtualDeviceConfigSpec()
					Expect(configSpec.Device.GetVirtualDevice().Key).To(Equal(expectedList[i-2].GetVirtualDevice().Key))
					Expect(configSpec.Operation).To(Equal(vimTypes.VirtualDeviceConfigSpecOperationAdd))
				}
			})
		})

		Context("When the expected and current list of pciDevices have same Devices", func() {
			BeforeEach(func() {
				currentList = append(currentList, vGPUDevice1)
				expectedList = append(expectedList, vGPUDevice1)
				currentList = append(currentList, dynamicDirectPathIODev1)
				expectedList = append(expectedList, dynamicDirectPathIODev1)
			})

			It("returns empty list", func() {
				Expect(err).ToNot(HaveOccurred())
				Expect(deviceChanges).To(BeEmpty())
			})
		})
	})
})
