package constants

import "time"

// Default public IP lookup (network_summary.public_ip).
const PublicIPLookupURL = "https://api.ipify.org?format=json"

const PublicIPLookupTimeout = 5 * time.Second
