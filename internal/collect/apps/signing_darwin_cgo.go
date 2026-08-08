//go:build darwin && cgo

package apps

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

static char *te_cfstring_copy(CFStringRef s) {
	if (s == NULL) {
		return NULL;
	}
	CFIndex len = CFStringGetLength(s);
	CFIndex max = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
	char *buf = (char *)malloc((size_t)max);
	if (buf == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(s, buf, max, kCFStringEncodingUTF8)) {
		free(buf);
		return NULL;
	}
	return buf;
}

static SecStaticCodeRef te_static_code_create(const char *path, OSStatus *outStatus) {
	if (path == NULL || *path == '\0') {
		if (outStatus) *outStatus = errSecParam;
		return NULL;
	}
	CFStringRef pathStr = CFStringCreateWithCString(kCFAllocatorDefault, path, kCFStringEncodingUTF8);
	if (pathStr == NULL) {
		if (outStatus) *outStatus = errSecAllocate;
		return NULL;
	}
	CFURLRef url = CFURLCreateWithFileSystemPath(kCFAllocatorDefault, pathStr, kCFURLPOSIXPathStyle, false);
	CFRelease(pathStr);
	if (url == NULL) {
		if (outStatus) *outStatus = errSecParam;
		return NULL;
	}
	SecStaticCodeRef code = NULL;
	OSStatus st = SecStaticCodeCreateWithPath(url, kSecCSDefaultFlags, &code);
	CFRelease(url);
	if (outStatus) *outStatus = st;
	if (st != errSecSuccess) {
		return NULL;
	}
	return code;
}

// te_extract_signing fills outIdentifier/outTeam/outSubject (caller frees).
// Returns OSStatus from SecCodeCopySigningInformation.
static OSStatus te_extract_signing(const char *path, char **outIdentifier, char **outTeam, char **outSubject) {
	if (outIdentifier) *outIdentifier = NULL;
	if (outTeam) *outTeam = NULL;
	if (outSubject) *outSubject = NULL;

	OSStatus st = errSecSuccess;
	SecStaticCodeRef code = te_static_code_create(path, &st);
	if (code == NULL) {
		return st;
	}

	CFDictionaryRef info = NULL;
	st = SecCodeCopySigningInformation(code, kSecCSSigningInformation | kSecCSRequirementInformation, &info);
	CFRelease(code);
	if (st != errSecSuccess || info == NULL) {
		if (info) CFRelease(info);
		return st;
	}

	if (outIdentifier) {
		CFStringRef ident = (CFStringRef)CFDictionaryGetValue(info, kSecCodeInfoIdentifier);
		*outIdentifier = te_cfstring_copy(ident);
	}
	if (outTeam) {
		CFStringRef team = (CFStringRef)CFDictionaryGetValue(info, kSecCodeInfoTeamIdentifier);
		*outTeam = te_cfstring_copy(team);
	}
	if (outSubject) {
		CFArrayRef certs = (CFArrayRef)CFDictionaryGetValue(info, kSecCodeInfoCertificates);
		if (certs != NULL && CFArrayGetCount(certs) > 0) {
			SecCertificateRef cert = (SecCertificateRef)CFArrayGetValueAtIndex(certs, 0);
			if (cert != NULL) {
				CFStringRef summary = SecCertificateCopySubjectSummary(cert);
				*outSubject = te_cfstring_copy(summary);
				if (summary) CFRelease(summary);
			}
		}
	}
	CFRelease(info);
	return errSecSuccess;
}

static OSStatus te_validate_signing(const char *path) {
	OSStatus st = errSecSuccess;
	SecStaticCodeRef code = te_static_code_create(path, &st);
	if (code == NULL) {
		return st;
	}
	st = SecStaticCodeCheckValidity(code, kSecCSDefaultFlags, NULL);
	CFRelease(code);
	return st;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type securitySigner struct{}

func newPlatformSigner() Signer {
	return securitySigner{}
}

func (securitySigner) Extract(path string) (SigningInfo, error) {
	if path == "" {
		return SigningInfo{}, fmt.Errorf("empty path")
	}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	var ident, team, subject *C.char
	st := C.te_extract_signing(cPath, &ident, &team, &subject)
	info := SigningInfo{}
	if ident != nil {
		info.SigningIdentifier = C.GoString(ident)
		C.free(unsafe.Pointer(ident))
	}
	if team != nil {
		info.TeamID = C.GoString(team)
		C.free(unsafe.Pointer(team))
	}
	if subject != nil {
		info.CertificateSubject = C.GoString(subject)
		C.free(unsafe.Pointer(subject))
	}
	if st != C.errSecSuccess {
		return info, fmt.Errorf("SecCodeCopySigningInformation: OSStatus %d", int(st))
	}
	return info, nil
}

func (securitySigner) Validate(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("empty path")
	}
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	st := C.te_validate_signing(cPath)
	if st == C.errSecSuccess {
		return true, nil
	}
	// Validation completed but signature is not valid.
	return false, nil
}
