#!/bin/bash
echo "Testing if strings.HasPrefix works..."

cat > /tmp/test_prefix.go << 'GOEOF'
package main
import (
"fmt"
"strings"
)
func main() {
token := "__TOOL_CALL__[{\"function\":{\"name\":\"read_file\"}}]"
fmt.Printf("Token: %s\n", token)
fmt.Printf("Has prefix: %v\n", strings.HasPrefix(token, "__TOOL_CALL__"))
fmt.Printf("Trimmed: %s\n", strings.TrimPrefix(token, "__TOOL_CALL__"))
}
GOEOF

go run /tmp/test_prefix.go
