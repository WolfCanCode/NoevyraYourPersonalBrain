package app

import (
 "bytes"
 "encoding/json"
 "strings"
 "testing"
 "github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

func TestMCPProtocolAndCandidatePolicy(t *testing.T){
 s,e:=memory.Open(t.TempDir());if e!=nil{t.Fatal(e)};defer s.Close()
 input:=strings.Join([]string{
  `null`,
  `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
  `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"future","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
  `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
  `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"brain_remember","arguments":{"content":"architecture candidate"}}}`,
  `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"brain_remember","arguments":{"content":"global","scope":"personal"}}}`,
  `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"brain_recall","arguments":{"query":"architecture","limit":0}}}`,
  `{"jsonrpc":"2.0","id":6,"method":"missing"}`,
  `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`,
  `{`,
 },"\n")
 var output bytes.Buffer
 if e=ServeMCP(s,"p","test",strings.NewReader(input),&output);e!=nil{t.Fatal(e)}
 lines:=strings.Split(strings.TrimSpace(output.String()),"\n");if len(lines)!=9{t.Fatalf("responses %d: %s",len(lines),output.String())}
 results:=[]map[string]any{};for _,line:=range lines{var v map[string]any;if e=json.Unmarshal([]byte(line),&v);e!=nil{t.Fatal(e)};results=append(results,v)}
 if results[0]["error"].(map[string]any)["code"]!=float64(-32600){t.Fatal("null must be invalid")}
 if results[1]["error"]==nil{t.Fatal("tools before initialization accepted")}
 if results[2]["result"].(map[string]any)["protocolVersion"]!="2025-06-18"{t.Fatal("unknown protocol blindly accepted")}
 if results[3]["result"].(map[string]any)["isError"]!=false{t.Fatal("candidate write failed")}
 for _,i:=range []int{4,5}{if results[i]["result"].(map[string]any)["isError"]!=true{t.Fatal("invalid tool call allowed")}}
 rows,e:=s.Recall("architecture","p",8);if e!=nil||len(rows)!=0{t.Fatal("unapproved memory leaked")}
 if results[8]["error"].(map[string]any)["code"]!=float64(-32700){t.Fatal("parse error not reported")}
}
func TestCLIFlagsDoNotSilentlyChangeMeaning(t *testing.T){
 for _,args:=range [][]string{{"--scope"},{"--bad"},{"--scope","personal","--scope","project"}}{if _,_,e:=flags(args,[]string{"--scope"},nil);e==nil{t.Fatalf("accepted %+v",args)}}
}
