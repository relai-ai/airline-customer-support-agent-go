package airline

import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "testing"
)

func TestRelaiTraceConcurrentRedaction(t *testing.T) {
 path := filepath.Join(t.TempDir(), "trajectory.json")
 t.Setenv("RELAI_ATIF_TRAJECTORY_PATH",path)
 t.Setenv("OPENAI_API_KEY","test-only-sensitive-value")
 recorder := newRelaiTrace()
 var wg sync.WaitGroup
 for i:=0; i<12; i++ { wg.Add(1); go func(){ defer wg.Done(); recorder.step("user", `observed test-only-sensitive-value Authorization: Bearer abc123`, "") }() }
 wg.Wait()
 data,err := os.ReadFile(path); if err != nil { t.Fatal(err) }
 if strings.Contains(string(data), "test-only-sensitive-value") || strings.Contains(string(data),"abc123") { t.Fatal("secret escaped") }
 var doc struct { Steps []struct { ID int `json:"step_id"` } `json:"steps"` }
 if err := json.Unmarshal(data,&doc); err != nil { t.Fatal(err) }
 if len(doc.Steps)!=12 { t.Fatal("lost steps") }
 for i,s := range doc.Steps { if s.ID!=i+1 { t.Fatal("out of order") } }
 matches,_ := filepath.Glob(filepath.Join(filepath.Dir(path),".atif-*")); if len(matches)!=0 { t.Fatal("temporary files leaked") }
}

func TestRelaiTraceDisabled(t *testing.T) {
 t.Setenv("RELAI_ATIF_TRAJECTORY_PATH","")
 if newRelaiTrace()!=nil { t.Fatal("recorder should be disabled") }
}
