//go:build windows

package collect

import (
	"testing"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
)

func TestParseWinSecurityArtifactsJSON(t *testing.T) {
	sample := `[
		{"kind":"driver","name":"WdFilter","display_name":"Microsoft Defender Antivirus Mini-Filter Driver","state":"Running","status":"OK","path":"C:\\Windows\\system32\\drivers\\WdFilter.sys","service_type":"File System Driver","start_mode":"System","account":null,"hive":null,"key_path":null,"value_name":null,"value":null},
		{"kind":"service","name":"UpdaterSvc","display_name":"Updater Service","state":"Stopped","status":"OK","path":"C:\\Updater\\updater.exe","service_type":"Own Process","start_mode":"Auto","account":"LocalSystem","hive":null,"key_path":null,"value_name":null,"value":null},
		{"kind":"registry","name":null,"display_name":null,"state":null,"status":null,"path":null,"service_type":null,"start_mode":null,"account":null,"hive":"HKCU","key_path":"Software\\Microsoft\\Windows\\CurrentVersion\\Run","value_name":"Updater","value":"C:\\Updater\\updater.exe"}
	]`

	artifacts, err := parseWinSecurityArtifactsJSON(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 3 {
		t.Fatalf("artifacts=%+v want 3", artifacts)
	}
	if artifacts[0].Type != constants.TypeDriverLoad {
		t.Fatalf("driver type=%s", artifacts[0].Type)
	}
	if artifacts[1].Type != constants.TypeServiceInstall {
		t.Fatalf("service type=%s", artifacts[1].Type)
	}
	if artifacts[2].Type != constants.TypeRegistryPersist {
		t.Fatalf("registry type=%s", artifacts[2].Type)
	}
	if artifacts[2].Payload["value_name"] != "Updater" {
		t.Fatalf("registry payload=%v", artifacts[2].Payload)
	}
}
