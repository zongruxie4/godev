package app

import "github.com/tinywasm/fmt"

// ormc:formonly
type stateResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      fmt.RawJSON `json:"id"`
	Result  fmt.RawJSON `json:"result"`
}
