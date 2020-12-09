package ng

import (
   "os"
   "fmt"
   "errors"
   "reflect"
   "strings"
   "io/ioutil"
//   "github.com/kr/pretty"
   "github.com/luisfurquim/goose"
   "github.com/gabrielledf/paperfishGo"
   "github.com/luisfurquim/stonelizard"
)

type Generator map[string]ModuleT

type OperationT struct {
   met string
   url string
   urlParm []string
   hdParm map[string]string
   qryParm map[string]string
   frmParm map[string]string
   bdParm map[string]string
   output map[string]map[string]struct{}
   outputset string
   outputvar string
   outputtype string
}

type ModuleT struct {
   op map[string]OperationT
   properties map[string]string
}

type GooseG struct {
   Gen goose.Alert
}

var Goose GooseG

var ErrUndefParam error = errors.New("Undefined parameter")

var decls map[string]string = map[string]string{}

var fperm os.FileMode = 0644
var dperm os.FileMode = 0755

var depends map[string]map[string]struct{} = map[string]map[string]struct{}{
   "@angular/common/http": map[string]struct{}{
      "HttpClient": struct{}{},
      "HttpErrorResponse": struct{}{},
   },

   "@angular/core": map[string]struct{}{
      "Injectable": struct{}{},
   },

   "rxjs": map[string]struct{}{
      "throwError": struct{}{},
   },

   "rxjs/operators": map[string]struct{}{
      "catchError": struct{}{},
   },

}

var typeDepends map[string]map[string]struct{} = map[string]map[string]struct{}{}

var types map[reflect.Type]string = map[reflect.Type]string{
   reflect.TypeOf(true)        : "boolean",
   reflect.TypeOf(int(0))      : "number",
   reflect.TypeOf(int8(0))     : "number",
   reflect.TypeOf(int16(0))    : "number",
   reflect.TypeOf(int32(0))    : "number",
   reflect.TypeOf(int64(0))    : "number",
   reflect.TypeOf(uint(0))     : "number",
   reflect.TypeOf(uint8(0))    : "number",
   reflect.TypeOf(uint16(0))   : "number",
   reflect.TypeOf(uint32(0))   : "number",
   reflect.TypeOf(uint64(0))   : "number",
   reflect.TypeOf(float32(0))  : "number",
   reflect.TypeOf(float64(0))  : "number",
   reflect.TypeOf("")          : "string",
   reflect.TypeOf([]byte{})    : "string",
   reflect.TypeOf([]bool{})    : "boolean[]",
   reflect.TypeOf([]int{})     : "number[]",
   reflect.TypeOf([]int8{})    : "number[]",
   reflect.TypeOf([]int16{})   : "number[]",
   reflect.TypeOf([]int32{})   : "number[]",
   reflect.TypeOf([]int64{})   : "number[]",
   reflect.TypeOf([]uint{})    : "number[]",
   reflect.TypeOf([]uint8{})   : "number[]",
   reflect.TypeOf([]uint16{})  : "number[]",
   reflect.TypeOf([]uint32{})  : "number[]",
   reflect.TypeOf([]uint64{})  : "number[]",
   reflect.TypeOf([]float32{}) : "number[]",
   reflect.TypeOf([]float64{}) : "number[]",
   reflect.TypeOf([]string{})  : "string[]",
   reflect.TypeOf([][]byte{})  : "string[]",
}

var predefinedTypes map[string]struct{} = map[string]struct{}{
   "boolean": struct{}{},
   "boolean[]": struct{}{},
   "string": struct{}{},
   "string[]": struct{}{},
   "number": struct{}{},
   "number[]": struct{}{},
}

func New() Generator {
   return Generator{}
}

func (ng Generator) Init(ws paperfishGo.WSClientT) {
   var modName string
   var propName string
   var mod map[string]paperfishGo.ModData
   var prop paperfishGo.ModData
   var t string
   var ok bool
   var err error

   for modName, mod = range ws.Modules {
      if _, ok = ng[modName] ; !ok {
         ng[modName] = ModuleT{
            op: map[string]OperationT{},
            properties: map[string]string{},
         }
      }
      for propName, prop = range mod {
         if t, ok = types[prop.Type]; !ok {
            t, _, err = registerType(modName, prop.Schema, prop.Type)
            if err!=nil {
               Goose.Gen.Fatalf(0,fmt.Sprintf("Error registering type %s: %s", prop.Type.Name(), err))
            }
         }
         ng[modName].properties[propName] = t
      }
   }
}

func (ng Generator) HandleOper(pfx, met string, opId string, op *paperfishGo.OperationT) {
   var err error
   var opPath string
   var ok bool
   var parm, resp *paperfishGo.ParameterT
   var newOp OperationT
   var t string
   var name string
//   var typ map[string]struct{}
//   var allTypes []string

   if op.XModule == "" {
      return
   }

   if _, ok = ng[op.XModule]; !ok {
      ng[op.XModule] = ModuleT{
         op: map[string]OperationT{},
         properties: map[string]string{},
      }
   }

//   opId = pfx + camel(met) + camel(opId)
//   opId = pfx + camel(opId)

   if _, ok = ng[op.XModule].op[opId]; !ok {
      newOp = OperationT{
         met: met,
         hdParm: map[string]string{},
         qryParm: map[string]string{},
         frmParm: map[string]string{},
         bdParm: map[string]string{},
         output: map[string]map[string]struct{}{},
      }
   } else {
      newOp = ng[op.XModule].op[opId]
   }

   opPath = "'/" + strings.SplitN(op.Path,"/",2)[1] + "'"
   for _, parm = range op.PathParm {
      if t, ok = types[parm.Type]; !ok {
         t, _, err = registerType(op.XModule, parm.Schema, parm.Type)
         if err!=nil {
            Goose.Gen.Fatalf(0,fmt.Sprintf("Error registering type %s: %s", parm.Name, err))
         }
      }
//      ng[op.XModule].properties[parm.Name] = t
//      opPath = strings.Replace(opPath, "{" + parm.Name + "}", "' + this." + pfx + met + opId + "." + parm.Name + " + '", -1)
      opPath = strings.Replace(opPath, "{" + parm.Name + "}", "' + encodeURIComponent(" + parm.Name + ") + '", -1)
      newOp.urlParm = append(newOp.urlParm, parm.Name)
   }
   newOp.url = opPath

   if len(op.HeaderParm)>0 {
      depends["@angular/common/http"]["HttpHeaders"] = struct{}{}
      for _, parm = range op.HeaderParm {
         if t, ok = types[parm.Type]; !ok {
            if parm.Type == nil {
               t = "string"
            } else {
               t, _, err = registerType(op.XModule, parm.Schema, parm.Type)
               if err!=nil {
                  Goose.Gen.Fatalf(0,fmt.Sprintf("Error registering type %s: %s", parm.Name, err))
               }
            }
         }
         newOp.hdParm[parm.Name] = t
//         ng[op.XModule].properties[parm.Name] = t
      }
   }

   if len(op.QueryParm)>0 {
      for _, parm = range op.QueryParm {
         if t, ok = types[parm.Type]; !ok {
            t, _, err = registerType(op.XModule, parm.Schema, parm.Type)
            if err!=nil {
               Goose.Gen.Fatalf(0,fmt.Sprintf("Error registering type %s: %s", parm.Name, err))
            }
         }
         newOp.qryParm[parm.Name] = t
//         ng[op.XModule].properties[parm.Name] = t
      }
   }

   if len(op.FormParm)>0 {
      for _, parm = range op.FormParm {
         if t, ok = types[parm.Type]; !ok {
            t, _, err = registerType(op.XModule, parm.Schema, parm.Type)
            if err!=nil {
               Goose.Gen.Fatalf(0,fmt.Sprintf("Error registering type %s: %s", parm.Name, err))
            }
         }
         newOp.frmParm[parm.Name] = t
//         ng[op.XModule].properties[parm.Name] = t
      }
   }

   if op.BodyParm != nil {
      if t, ok = types[op.BodyParm.Type]; !ok {
         t, _, err = registerType(op.XModule, op.BodyParm.Schema, op.BodyParm.Type)
         if err!=nil {
            Goose.Gen.Fatalf(0,fmt.Sprintf("Error registering type %s: %s", op.BodyParm.Name, err))
         }
      }
      newOp.bdParm[op.BodyParm.Name] = t
//      ng[op.XModule].properties[op.BodyParm.Name] = t
   }

   if len(op.Output) > 0 {
      newOp.outputvar = op.XOutputVar
      newOp.outputset = op.XOutput
      Goose.Gen.Logf(0,"op.Output: %#v", op.Output)
      for _, resp = range op.Output {
         if op.XOutputVar != "" {
            name = op.XOutputVar
         } else {
            name = resp.Name
         }
         Goose.Gen.Logf(2,"opId: %s", opId)
//         if opId == "wsGetGetPedido" {
//            Goose.Gen.Fatalf(0,"resp: %#v", resp)
//         }
         if t, ok = types[resp.Type]; !ok {
            t, _, err = registerType(op.XModule, resp.Schema, resp.Type)
            if err!=nil {
               Goose.Gen.Fatalf(0,fmt.Sprintf("Error registering type %s: %s", name, err))
            }
         }
         if _, ok = newOp.output[name]; ok {
            newOp.output[name][t] = struct{}{}
         } else {
            newOp.output[name] = map[string]struct{}{
               t: struct{}{},
            }
         }
      }

/*
      for name, typ = range newOp.output {
         allTypes = []string{}
         for t, _ = range typ {
            allTypes = append(allTypes, t)
         }
         ng[op.XModule].properties[name] = strings.Join(allTypes," | ")
      }
*/

   }
   ng[op.XModule].op[opId] = newOp
}

func (ng Generator) GenerateTypes(dir, pfx string, ws paperfishGo.WSClientT) {
   var err error
   var pkg string
   var imports string
   var typdef string
   var typeName string

   for typeName, typdef = range decls {
      imports = ""
      for pkg, _ = range typeDepends[typeName] {
         imports += "import { " + pkg + " } from '" + dir + "/" + pkg + "';\n"
      }
      if imports != "" {
         imports += "\n"
      }
      err = ioutil.WriteFile(fmt.Sprintf("%s%c%s.ts", dir, os.PathSeparator, typeName), []byte(imports + typdef), fperm)
      if err != nil {
         Goose.Gen.Fatalf(0,"Error creating %s: %s", fmt.Sprintf("%s.ts", typeName), err)
      }
   }

}

func (ng Generator) GenerateClients(dir, pfx, tdir string, ws paperfishGo.WSClientT) {
   var err error
   var modName string
   var modDef ModuleT
   var svcCode string
   var imports string
   var opId string
   var p string
   var option string
   var options []string
   var opt []string
   var propName, propType string
   var opDef OperationT
   var bdy []string
   var handleError string
   var metParm []string
   var outputType string
   var symbols map[string]struct{}
   var pkg string

   for modName, modDef = range ng {
      if modName == "" {
         continue
      }

      err = os.MkdirAll(fmt.Sprintf("%s%c%s", dir, os.PathSeparator, modName), 0700)
      if err != nil {
         Goose.Gen.Fatalf(0,fmt.Sprintf("Error creating client module directory %s: %s", modName, err))
      }

      imports = ""
      for pkg, symbols = range depends {
         imports += "import { " + join(symbols) + " } from '" + pkg + "';\n"
      }

      svcCode = "\n@Injectable({\n  providedIn: 'root'\n})\nexport class " + modName + "WS {\n\tconstructor(private http: HttpClient) {}\n"

      for propName, propType = range modDef.properties {
         svcCode += "\tpublic " + propName + ": " + propType + ";\n"
         imports += "import { " + propType + " } from '" + modName + ".service';\n"
         Goose.Gen.Fatalf(0,"imports: %s", imports)
      }

      for opId, opDef = range modDef.op {
         for outputType, _ = range opDef.output[opDef.outputvar] {
            imports += "import { " + outputType + " } from '" + tdir + "/" + outputType + "';\n"
         }
      }

      svcCode = imports + svcCode

      for opId, opDef = range modDef.op {
         metParm = opDef.urlParm[:]

         for outputType, _ = range opDef.output[opDef.outputvar] {
            svcCode += "\n\tpublic " + opDef.outputvar + ": " + outputType + ";\n"
         }

//         Goose.Gen.Fatalf(0,"outputvar: %s, output: %#v", opDef.outputvar, opDef.output)

         svcCode += "\n\t" + camel(opId) + "(%<<<metParm>>>) {\n" +
            "\t\tthis.http." + opDef.met + `<` + outputType + ">(" + opDef.url
//            "\t\tthis.http." + opDef.met + `<` + modDef.properties[opDef.outputvar] + ">(" + opDef.url

         if (opDef.met=="post" || opDef.met=="put") && (opDef.bdParm!=nil) {
            bdy = []string{}
            for propName, _ = range opDef.bdParm {
               bdy = append(bdy, "JSON.stringify(" + propName + ")")
               metParm = append(metParm, propName)
            }
            svcCode += ", " + strings.Join(bdy," + ")
         }

         if len(opDef.hdParm) > 0 {
            option  = "'headers':{"
            opt = []string{}
            for p, _ = range opDef.hdParm {
               opt = append(opt,p + ": " + p)
               metParm = append(metParm, p)
            }
            option += strings.Join(opt,",") + " }"
            options = append(options, option)
         }

         if len(opDef.qryParm) > 0 {
            option  = "'params':new HttpParams({ fromObject: {"
            opt = []string{}
            for p, _ = range opDef.qryParm {
               opt = append(opt,p + ": " + p)
               metParm = append(metParm, p)
            }
            option += strings.Join(opt,",") + " } })"
            options = append(options, option)
         }

         if len(options) > 0 {
            svcCode += ", {" + strings.Join(options, ", ") + "}"
         }

         svcCode = strings.Replace(svcCode, "%<<<metParm>>>", strings.Join(metParm,",") + ",onSuccess",-1)

         svcCode += ")\n\t\t.pipe(\n\t\t\tcatchError(this.handleError)\n\t\t)" +
                    ".subscribe(" + opDef.outputvar + " => {\n\t\t\t" +
                    strings.Replace(opDef.outputset, "\\n", "\n", -1) +
                    "\n\t\t\tif (onSuccess!==undefined) {" +
                    "\n\t\t\t\tonSuccess();" +
                    "\n\t\t\t}" +
                    "\n\t\t});\n\t}\n"

         if handleError == "" {

            handleError = `
   private handleError(error: HttpErrorResponse) {
      var local;
      if (error.error instanceof ErrorEvent) {
         // A client-side or network error occurred. Handle it accordingly.
         console.error('An error occurred:', error.error.message);
         local = 'navegador';
      } else {
         // The backend returned an unsuccessful response code.
         // The response body may contain clues as to what went wrong,
         console.error(
            ` + "`Backend returned code ${error.status}, `" + ` +
            ` + "`body was: ${error.error}`" + `);
         if (error.status<500) {
            local = 'navegador';
         } else {
            local = 'servidor';
         }
      }
      // return an observable with a user-facing error message
      return throwError(` + "`Erro no ${local}; tente novamente mais tarde.`" + `);
   };
`
         }
      }

      // File Saving
      err = ioutil.WriteFile(fmt.Sprintf("%s%c%s%c%s.module.ts", dir, os.PathSeparator, modName, os.PathSeparator, modName), []byte(`import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { BrowserModule } from '@angular/platform-browser';
import { HttpClientModule } from '@angular/common/http';


@NgModule({
   imports: [
      BrowserModule,
      HttpClientModule,
      CommonModule
   ]
})
   `), fperm)
      if err != nil {
         Goose.Gen.Fatalf(0,fmt.Sprintf("Error creating %s%c%s.module.ts: %s", modName, os.PathSeparator, modName, err))
      }

      err = ioutil.WriteFile(fmt.Sprintf("%s%c%s%c%s.service.ts", dir, os.PathSeparator, modName, os.PathSeparator, modName), []byte(svcCode+handleError+"\n};\n"), fperm)
      if err != nil {
         Goose.Gen.Fatalf(0,fmt.Sprintf("Error creating %s%c%s.service.ts: %s", modName, os.PathSeparator, modName), err)
      }

   }

//   Goose.Gen.Logf(1,fmt.Sprintf("%s:%s -> %#v", met, opId, op))
}

func registerType(mod string, parm *stonelizard.SwaggerSchemaT, t reflect.Type) (string, reflect.Type, error) {
   var typ reflect.Type
   var ok bool
   var tname string
   var item string
   var err error
   var i int
   var fld reflect.StructField
   var fieldDecls, fldType string
   var schema stonelizard.SwaggerSchemaT
   var finalName string
   var subDependents map[string]struct{}
   var dependsOn string

   switch t.Kind() {
   case reflect.Array:
      if item, ok = types[t.Elem()]; !ok  { // mudar de name para parameter e pegar o nome do tipo
         Goose.Gen.Logf(1,fmt.Sprintf("part: %#v", parm))
         item, _, err = registerType(mod, parm.Items, t.Elem())
         if err!=nil {
            return "", t, err
         }
      }
      tname = item + "[]"
      if _, ok = types[t]; !ok {
         types[t] = tname
      }

   case reflect.Map:
      if item, ok = types[t.Elem()]; !ok  {
         item, _, err = registerType(mod, parm.Items, t.Elem())
         if err!=nil {
            return "", t, err
         }
      }
      if _, ok = decls["Dictionary<>"]; !ok {
         decls["Dictionary<>"] = `
export class Dictionary<T> {
    [key: string]: T;
}
`
      }
      tname = "Dictionary<" + item + ">";
      if _, ok = types[t]; !ok {
         return "", t, err
      }

   case reflect.Struct:
      Goose.Gen.Logf(1,fmt.Sprintf("t.NumField(): %d", t.NumField()))
      subDependents = map[string]struct{}{}
      for i=0; i<t.NumField(); i++ {
         Goose.Gen.Logf(1,fmt.Sprintf("i: %d", i))
         fld = t.Field(i)
         finalName = strings.Split(fld.Tag.Get("json"),",")[0]
         Goose.Gen.Logf(1,fmt.Sprintf("json name: %s", finalName))
         if finalName == "" {
            finalName = fld.Name
         }
         if fldType, ok = types[fld.Type]; !ok  && parm!=nil {
            if parm == nil {
               Goose.Gen.Logf(1,fmt.Sprintf("mod: %s, t: %#v", mod, t))
               return "", t, ErrUndefParam
            }

/*
            Goose.Gen.Logf(1,fmt.Sprintf("fld.Name: %s  -  finalName: %s", fld.Name, finalName))
            for prop, _ := range parm.Properties {
               Goose.Gen.Logf(1,fmt.Sprintf("prop: %s", prop))
            }
*/

            Goose.Gen.Logf(1,fmt.Sprintf("fld.Name: %s\nparm: %#v", finalName, parm))
            schema = parm.Properties[finalName]
            fldType, _, err = registerType(mod, &schema, fld.Type)
            if err!=nil {
               return "", t, err
            }
         }
         fieldDecls += "\t" + finalName + ": " + fldType + ";\n"
         Goose.Gen.Logf(1,fmt.Sprintf("fieldDecls: %s", fieldDecls))
         dependsOn = types[fld.Type]
         if _, ok = predefinedTypes[dependsOn]; !ok {
            if dependsOn[len(dependsOn)-1] == ']' {
               dependsOn = dependsOn[:len(dependsOn)-2]
            }
            if _, ok = subDependents[dependsOn]; !ok {
               subDependents[dependsOn] = struct{}{}
            }
         }
      }
      Goose.Gen.Logf(1,fmt.Sprintf("parm: %#v", parm))
      Goose.Gen.Logf(1,fmt.Sprintf("parm.Title: %s", parm.Title))
      tname = parm.Title

      if _, ok = decls[tname]; !ok {
         decls[tname] = "export class " + tname + " {\n" + fieldDecls + "}\n"
      }
      if _, ok = types[t]; !ok {
         types[t] = tname
      }
      if _, ok = typeDepends[tname]; !ok && len(subDependents)>0 {
         typeDepends[tname] = subDependents
      }
   }

   return tname, typ, nil
}

func camel(s string) string {
   if len(s)==0 {
      return s
   }

   return strings.ToUpper(s[:1]) + s[1:]
}

func join(m map[string]struct{}) string {
   var s string
   var k string

   for k = range m {
      s += k + ", "
   }

   return s[:len(s)-2]
}




//
// no maindir gerar declaração de tipos -> decls
// depois criar um dir para cada modulo
//
