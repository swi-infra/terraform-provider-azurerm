package compute

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

type VirtualMachineID struct {
	ResourceGroup string
	Name          string
}

func ParseVirtualMachineID(input string) (*VirtualMachineID, error) {
	id, err := azure.ParseAzureResourceID(input)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Unable to parse Virtual Machine ID %q: %+v", input, err)
	}

	virtualMachine := VirtualMachineID{
		ResourceGroup: id.ResourceGroup,
	}

	if virtualMachine.Name, err = id.PopSegment("virtualMachines"); err != nil {
		return nil, err
	}

	if err := id.ValidateNoEmptySegments(input); err != nil {
		return nil, err
	}

	return &virtualMachine, nil
}

func ValidateVirtualMachineID(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("expected type of %q to be string", k))
		return
	}

	if _, err := ParseVirtualMachineID(v); err != nil {
		errors = append(errors, fmt.Errorf("Can not parse %q as a resource id: %v", k, err))
		return
	}

	return warnings, errors
}

func expandVirtualMachineNetworkInterfaceIDs(input []interface{}) []compute.NetworkInterfaceReference {
	output := make([]compute.NetworkInterfaceReference, 0)

	for _, v := range input {
		output = append(output, compute.NetworkInterfaceReference{
			ID: utils.String(v.(string)),
		})
	}

	return output
}

func flattenVirtualMachineNetworkInterfaceIDs(input *[]compute.NetworkInterfaceReference) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	output := make([]interface{}, 0)

	for _, v := range *input {
		if v.ID == nil {
			continue
		}

		output = append(output, *v.ID)
	}

	return output
}

func VirtualMachineOSDiskSchema() *schema.Schema {
	// TODO: for a Virtual Machine some of these properties should be ForceNew
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"caching": {
					Type:     schema.TypeString,
					Required: true,
					ValidateFunc: validation.StringInSlice([]string{
						string(compute.CachingTypesNone),
						string(compute.CachingTypesReadOnly),
						string(compute.CachingTypesReadWrite),
					}, false),
				},
				"storage_account_type": {
					Type:     schema.TypeString,
					Required: true,
					// whilst this appears in the Update block the API returns this when changing:
					// Changing property 'osDisk.managedDisk.storageAccountType' is not allowed
					ForceNew: true,
					ValidateFunc: validation.StringInSlice([]string{
						// note: OS Disks don't support Ultra SSDs
						string(compute.StorageAccountTypesPremiumLRS),
						string(compute.StorageAccountTypesStandardLRS),
						string(compute.StorageAccountTypesStandardSSDLRS),
					}, false),
				},

				// Optional
				"diff_disk_settings": {
					Type:     schema.TypeList,
					Optional: true,
					ForceNew: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"option": {
								Type:     schema.TypeString,
								Required: true,
								ForceNew: true,
								ValidateFunc: validation.StringInSlice([]string{
									string(compute.Local),
								}, false),
							},
						},
					},
				},

				"disk_encryption_set_id": {
					Type:     schema.TypeString,
					Optional: true,
					// TODO: make this more specific
					ValidateFunc: azure.ValidateResourceID,
				},

				"disk_size_gb": {
					Type:         schema.TypeInt,
					Optional:     true,
					Computed:     true,
					ValidateFunc: validation.IntBetween(0, 1023),
				},

				"name": {
					Type:     schema.TypeString,
					Optional: true,
					ForceNew: true,
					Computed: true,
				},

				"write_accelerator_enabled": {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	}
}

func ExpandVirtualMachineOSDisk(input []interface{}, osType compute.OperatingSystemTypes) *compute.OSDisk {
	raw := input[0].(map[string]interface{})
	disk := compute.OSDisk{
		Caching: compute.CachingTypes(raw["caching"].(string)),
		ManagedDisk: &compute.ManagedDiskParameters{
			StorageAccountType: compute.StorageAccountTypes(raw["storage_account_type"].(string)),
			// TODO: Disk Encryption Set ID
		},
		WriteAcceleratorEnabled: utils.Bool(raw["write_accelerator_enabled"].(bool)),

		// these have to be hard-coded so there's no point exposing them
		CreateOption: compute.DiskCreateOptionTypesFromImage,
		OsType:       osType,
	}

	if osDiskSize := raw["disk_size_gb"].(int); osDiskSize > 0 {
		disk.DiskSizeGB = utils.Int32(int32(osDiskSize))
	}

	if diffDiskSettingsRaw := raw["diff_disk_settings"].([]interface{}); len(diffDiskSettingsRaw) > 0 {
		diffDiskRaw := diffDiskSettingsRaw[0].(map[string]interface{})
		disk.DiffDiskSettings = &compute.DiffDiskSettings{
			Option: compute.DiffDiskOptions(diffDiskRaw["option"].(string)),
		}
	}

	if id := raw["disk_encryption_set_id"].(string); id != "" {
		disk.ManagedDisk.DiskEncryptionSet = &compute.DiskEncryptionSetParameters{
			ID: utils.String(id),
		}
	}

	if name := raw["name"].(string); name != "" {
		disk.Name = utils.String(name)
	}

	return &disk
}

func FlattenVirtualMachineOSDisk(input *compute.OSDisk) []interface{} {
	if input == nil {
		return []interface{}{}
	}

	diffDiskSettings := make([]interface{}, 0)
	if input.DiffDiskSettings != nil {
		diffDiskSettings = append(diffDiskSettings, map[string]interface{}{
			"option": string(input.DiffDiskSettings.Option),
		})
	}

	diskSizeGb := 0
	if input.DiskSizeGB != nil && *input.DiskSizeGB != 0 {
		diskSizeGb = int(*input.DiskSizeGB)
	}

	var name string
	if input.Name != nil {
		name = *input.Name
	}

	diskEncryptionSetId := ""
	storageAccountType := ""
	if input.ManagedDisk != nil {
		storageAccountType = string(input.ManagedDisk.StorageAccountType)

		if input.ManagedDisk.DiskEncryptionSet != nil && input.ManagedDisk.DiskEncryptionSet.ID != nil {
			diskEncryptionSetId = *input.ManagedDisk.DiskEncryptionSet.ID
		}
	}

	writeAcceleratorEnabled := false
	if input.WriteAcceleratorEnabled != nil {
		writeAcceleratorEnabled = *input.WriteAcceleratorEnabled
	}
	return []interface{}{
		map[string]interface{}{
			"caching":                   string(input.Caching),
			"disk_size_gb":              diskSizeGb,
			"diff_disk_settings":        diffDiskSettings,
			"disk_encryption_set_id":    diskEncryptionSetId,
			"name":                      name,
			"storage_account_type":      storageAccountType,
			"write_accelerator_enabled": writeAcceleratorEnabled,
		},
	}
}

func setVirtualMachineConnectionInformation(d *schema.ResourceData, input *compute.VirtualMachineProperties) {
	if input == nil {
		return
	}

	provisionerType := "ssh"
	if profile := input.OsProfile; profile != nil {
		if profile.WindowsConfiguration != nil {
			provisionerType = "winrm"
		}
	}

	// TODO: determine the public ip
	ipAddress := "1.2.3.4"
	d.SetConnInfo(map[string]string{
		"type": provisionerType,
		"host": ipAddress,
	})
}
