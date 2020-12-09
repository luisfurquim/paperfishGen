package main


import (
   "io"
   "os"
   "fmt"
   "flag"
   "regexp"
   "path/filepath"
   "github.com/luisfurquim/goose"
   "github.com/gabrielledf/paperfishGo"
   "github.com/luisfurquim/paperfishGen/ng"
)

type Generator interface {
   Init(ws paperfishGo.WSClientT)
   HandleOper(pfx, met string, opId string, op *paperfishGo.OperationT)
   GenerateTypes(dir, prefix string, ws paperfishGo.WSClientT)
   GenerateClients(dir, prefix, svcName string, ws paperfishGo.WSClientT)
}

type GooseG struct {
   Gen goose.Alert
}

var Goose GooseG
var version string = "0.1"
var reInput *regexp.Regexp = regexp.MustCompile(`^https?:\/\/`)
var reOper *regexp.Regexp = regexp.MustCompile(`^(?i:((?:get)|(?:post)|(?:put)|(?:delete)|(?:options)|(?:head)|(?:patch)):([a-z09_\$@]+))$`)

func main() {
   var defaultDebugLevel int
   var in string
   var pfx string
   var ver bool
   var ws []paperfishGo.WSClientT
   var fh io.Reader
   var err error
   var format string
   var gen Generator
   var opId string
   var opDef *paperfishGo.OperationT
   var Geese goose.Geese
   var bdir, tdir, cdir string

   flag.IntVar(&defaultDebugLevel, "v", 2, "Verbose level, 2 if omitted")
   flag.StringVar(&in, "in", "", "Source contract may be a file or a URL")
   flag.StringVar(&pfx, "pfx", "ws", "Prefix string to prepend on all service names")
   flag.StringVar(&bdir, "dir", "./", "Base project directory")
   flag.StringVar(&tdir, "tdir", "./", "Directory where all the types shoud be generated, relative to base dir")
   flag.StringVar(&cdir, "cdir", "./", "Directory where all the clients shoud be generated, relative to base dir")
   flag.StringVar(&format, "fmt", "ng", "Output format, currently only the default (ng==angular)")
   flag.BoolVar(&ver, "version", false, "Print version number")
   flag.Parse()

   goose.TraceOn()

   Geese = goose.Geese{
      "paperfishGo": &paperfishGo.Goose,
      "ng": &ng.Goose,
      "paperfishGen": &Goose,
   }
   Geese.Set(defaultDebugLevel)

   tdir, err = filepath.Abs(tdir)
   if err != nil {
      Goose.Gen.Fatalf(0,"Error normalizing types directory %s: %s", tdir, err)
   }

   cdir, err = filepath.Abs(cdir)
   if err != nil {
      Goose.Gen.Fatalf(0,"Error normalizing clients directory %s: %s", cdir, err)
   }

   bdir, err = filepath.Abs(bdir)
   if err != nil {
      Goose.Gen.Fatalf(0,"Error normalizing clients directory %s: %s", cdir, err)
   }

   if len(cdir)<=len(bdir) || len(tdir)<=len(bdir) || cdir[:len(bdir)] != bdir || tdir[:len(bdir)] != bdir {
      Goose.Gen.Fatalf(0,"Error invalid directory setup")
   }

   tdir = tdir[len(bdir):]
   if tdir[0] == os.PathSeparator {
      tdir = tdir[1:]
   }
   cdir = cdir[len(bdir):]
   if cdir[0] == os.PathSeparator {
      cdir = cdir[1:]
   }

   switch format {
   case "ng":
      gen = ng.New()
   }

   if reInput.MatchString(in) {
      ws, err = paperfishGo.NewFromURL(in, nil)
   } else {
      fh, err = os.Open(in)
      if err != nil {
         Goose.Gen.Fatalf(0,fmt.Sprintf("Error opening contract from %s: %s", in, err))
      }

      ws, err = paperfishGo.NewFromReader(fh, nil)
   }
   if err != nil {
      Goose.Gen.Fatalf(0,fmt.Sprintf("Error fetching contract from %s: %s", in, err))
   }

   err = os.MkdirAll(tdir, 0700)
   if err != nil {
      Goose.Gen.Fatalf(0,fmt.Sprintf("Error creating types directory %s: %s", tdir, err))
   }

   err = os.MkdirAll(cdir, 0700)
   if err != nil {
      Goose.Gen.Fatalf(0,fmt.Sprintf("Error creating clients directory %s: %s", cdir, err))
   }

   err = os.Chdir(bdir)
   if err != nil {
      Goose.Gen.Fatalf(0,fmt.Sprintf("Error going to base directory %s: %s", bdir, err))
   }

   gen.Init(ws[0])

   for opId, opDef = range ws[0].GetOperation {
      Goose.Gen.Logf(2,"%s: %#v", opId, opDef)
      gen.HandleOper(pfx, "get", opId, opDef)
   }

   for opId, opDef = range ws[0].PostOperation {
      Goose.Gen.Logf(2,"%s: %#v", opId, opDef)
      gen.HandleOper(pfx, "post", opId, opDef)
   }

   for opId, opDef = range ws[0].PutOperation {
      Goose.Gen.Logf(2,"%s: %#v", opId, opDef)
      gen.HandleOper(pfx, "put", opId, opDef)
   }

   for opId, opDef = range ws[0].DeleteOperation {
      Goose.Gen.Logf(2,"%s: %#v", opId, opDef)
      gen.HandleOper(pfx, "delete", opId, opDef)
   }

   for opId, opDef = range ws[0].OptionsOperation {
      Goose.Gen.Logf(2,"%s: %#v", opId, opDef)
      gen.HandleOper(pfx, "options", opId, opDef)
   }

   for opId, opDef = range ws[0].HeadOperation {
      Goose.Gen.Logf(2,"%s: %#v", opId, opDef)
      gen.HandleOper(pfx, "head", opId, opDef)
   }

   for opId, opDef = range ws[0].PatchOperation {
      Goose.Gen.Logf(2,"%s: %#v", opId, opDef)
      gen.HandleOper(pfx, "patch", opId, opDef)
   }

   gen.GenerateTypes(tdir, pfx, ws[0])
   gen.GenerateClients(cdir, pfx, tdir, ws[0])

}


