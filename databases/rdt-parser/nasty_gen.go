package main

// Last modified at 2026/02/11 星期三 22:25:54

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	mapset "github.com/deckarep/golang-set"
	"github.com/gobeam/stringy"
)

var (
	outerFormMap = map[string][]string{}
	primKeyMap   = map[string]string{}
	tsUpdater    = map[string]string{}
	tsCreator    = map[string]string{}
)

//go:generate go run .
func main() {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		_, _ = fmt.Fprintf(
			os.Stderr,
			"nasty-gen.main-defer: encounter an error: %v",
			err,
		)
	}()

	srcFileArr, err := getRDTfilesInDir("../rdt-def")
	if err != nil {
		panic(err)
	}
	for _, srcFile := range srcFileArr {
		listener := ResolveRDT(srcFile)
		if listener == nil {
			continue
		}
		srcFile = filepath.Base(srcFile)
		splitPos := strings.Index(filepath.Base(srcFile), ".")
		switch splitPos {
		case -1:
			splitPos = len(srcFile)
		case 0:
			srcFile = "aux_gen.go"
			splitPos = len(srcFile)
		}
		RDTGenAux(listener, "../"+srcFile[:splitPos]+"_gen.go")
	}
}

// Requirements:
// 	1. counter for some statistics 				(o)
// 	2. context for auto-recoding foreign keys 	(~)
// 	3. generic for all database backends		(?, partially)
// these three requirements should be satisfied at the same time.

type RdtListener struct {
	*BaseRDTparserListener
	MetaStruct       map[string]map[string]any
	ShouldImportTime bool
}

// ResolveRDT takes given file as input and generate RDT-AST listener as output.
func ResolveRDT(filePath string) *RdtListener {
	filestream, err := antlr.NewFileStream(filePath)
	if err != nil {
		return nil
	}
	lexer := NewRDTlexer(filestream)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewRDTparser(stream)
	parser.BuildParseTrees = true
	ast := parser.Source_file()
	listener := &RdtListener{}
	listener.MetaStruct = make(map[string]map[string]any)
	walker := antlr.NewParseTreeWalker()
	walker.Walk(listener, ast)
	return listener
}

// EnterTables hook function for code generation by walking in the AST provided by antlr4.
func (r *RdtListener) EnterTables(c *TablesContext) {
	tabName := c.Table_name().GetText()
	r.MetaStruct[tabName] = map[string]any{}
	for _, fieldInfo := range c.AllAttr_tuples() {
		fieldName := fieldInfo.Field_name().GetText()
		var tmp []IField_attrContext
		if fieldInfo.Field_attrs() != nil {
			tmp = fieldInfo.Field_attrs().AllField_attr()
		} else {
			tmp = nil
		}
		r.MetaStruct[tabName][fieldName] = map[string]any{
			"ty":   fieldInfo.Field_type().GetText(),
			"attr": r.astAttributesAux(tmp),
		}
	}
}

// astAttributesAux will set RdtListener's MetaStruct as long as in-param x has value.
func (r *RdtListener) astAttributesAux(x []IField_attrContext) map[string]any {
	var res = map[string]any{}
	for _, fieldAttr := range x {
		if fieldAttr.PRIMARY() != nil {
			res["primary"] = struct{}{}
		} else if fieldAttr.NOTNULL() != nil {
			res["notnull"] = struct{}{}
		} else if fieldAttr.UNIQUE() != nil {
			res["unique"] = struct{}{}
		} else if fieldAttr.AUTOINCREMENT() != nil {
			res["autoincrement"] = struct{}{}
		} else if fieldAttr.COUNTER() != nil {
			_, ok := res["counter"]
			if ok {
				panic("only allow one counter in one field")
			}
			res["counter"] = fieldAttr.INT_NUMBER().GetText()
		} else if fieldAttr.FOREIGN() != nil {
			outerTabName := fieldAttr.Table_name().GetText()
			res["foreign"] = map[string]string{
				outerTabName: fieldAttr.Field_name().GetText(),
			}
		} else if fieldAttr.CREATETIME() != nil {
			res["createtime"] = fieldAttr.Time_unit().GetText()
			r.ShouldImportTime = true

		} else if fieldAttr.UPDATETIME() != nil {
			res["updatetime"] = fieldAttr.Time_unit().GetText()
			r.ShouldImportTime = true

		} else if fieldAttr.INDEX() != nil {
			res["index"] = fieldAttr.Idx_name().GetText()
		} else if fieldAttr.DEFAULT() != nil {
			if fieldAttr.Literal() != nil {
				res["default"] = fieldAttr.Literal().GetText()
			} else {
				res["default"] = fieldAttr.INT_NUMBER().GetText()
			}
		} else {
			baseStr := fmt.Sprintf(
				"unknown token was found: %v",
				fieldAttr.GetText(),
			)
			panic(baseStr)
		}
	}
	return res
}

func getRDTfilesInDir(dir string) ([]string, error) {
	var files []string
	auxFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".rdt" {
			files = append(files, path)
		}
		return nil
	}
	err := filepath.WalkDir(dir, auxFn)
	return files, err
}

type (
	fieldInfo struct {
		/* keep this until definition needs extending */
		Type, Gtag, Json, Bson string
	}
	fieldRec map[string]*fieldInfo // str -> &{ str, str, str, str}
	fieldArr []fieldRec
	formRec  map[string]fieldArr // str -> [](str -> &{ str, str, str, str}, ...)
)

func (f fieldArr) Len() int {
	return len(f)
}

func (f fieldArr) Less(i, j int) bool {
	var field1, field2 = f[i], f[j]
	var f1Ty, f2Ty string
	for _, v := range field1 {
		f1Ty = v.Type
	}
	for _, v := range field2 {
		f2Ty = v.Type
	}
	// reverse the order so we get a decreasing sequence,
	// thus making the cast of memory alignment lower as much as possible
	return orderNum(f1Ty) > orderNum(f2Ty)
}

func (f fieldArr) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// TopUpperCamelConvertor converts any string into upper camel form.
func TopUpperCamelConvertor(x string) string {
	x = stringy.New(x).CamelCase().Get()
	if len(x) == 0 {
		panic("an empty string is passed into TopUpperCamelConvertor")
	}
	return strings.ToUpper(x[:1]) + x[1:]
}

// fieldNameSnakeConvertor converts any string into upper Snake form.
func fieldNameSnakeConvertor(x string) string {
	return stringy.New(x).SnakeCase().Get()
}

// LowerSnakeConvertor converts any string into lower Snake form.
func LowerSnakeConvertor(x string) string {
	x = fieldNameSnakeConvertor(x)
	return strings.ToLower(x)
}

// resolveRDTtype resolves basic RDT datatype.
func resolveRDTtype(x string) string {
	switch x {
	case "text":
		return "string"
	case "integer":
		return "int64"
	case "bytes":
		return "[]byte"
	default:
		return ""
	}
}

// orderNum roughly calculates used bytes of datatype in golang
func orderNum(x string) uint64 {
	if strings.Contains(x, "8") {
		return 0 // 0x1
	} else if strings.Contains(x, "16") {
		return 1 // 0x2
	} else if strings.Contains(x, "32") {
		return 2 // 0x4
	} else if strings.Contains(x, "64") {
		return 3 // 0x8
	} else if strings.Contains(x, "string") {
		return 4 // 0x16
	} else if strings.Contains(x, "[]") {
		return 5 // 0x18
	}
	// [FIX-THIS]: except for struct { only one basic-type. }
	return 6 // maybe tiny, or maybe excessive.
}

// isAttrForEasiestStructTag judges whether string `x` is the most basic type that database maintains
func isAttrForEasiestStructTag(x string) bool {
	return x == "primary" || x == "notnull" ||
		x == "unique" || x == "autoincrement"
}

// attrConvertor converts the string representation of attribute into golang structure tag.
func attrConvertor(x string) string {
	switch x {
	case "primary":
		return "primaryKey;"
	case "notnull":
		return "not null;"
	case "unique":
		return "unique;"
	case "autoincrement":
		return "autoIncrement;"
	default:
		return ""
	}
}

// timeUnitConvertor converts time unit for database structure tag in golang.
func timeUnitConvertor(x string) string {
	switch x {
	case "ns":
		return "nano"
	case "us":
		return "micro"
	case "ms":
		return "milli"
	default:
		return ""
	}
}

func RDTGenAux(
	listener *RdtListener,
	outputName string,
) {
	fd, err := os.Create(outputName)
	if err != nil {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"the stage of generating code encounter an error: %v", err,
		)
		return
	}
	defer func() { _ = fd.Close() }()
	os.Stdout = fd
	res := formRec{}
	auxFunc := mapset.NewSet()
	for tabName, tabMemArr := range listener.MetaStruct {
		tabName = TopUpperCamelConvertor(tabName)
		res[tabName] = make(fieldArr, 0)
		for fieldName, v := range tabMemArr {
			fap := fieldAttrProcessor{}
			fap.astListener = listener
			fap.v = v.(map[string]any)
			fap.tabName = tabName
			fap.fieldName = TopUpperCamelConvertor(fieldName)
			fap.snakeFieldName = LowerSnakeConvertor(fap.fieldName)
			fap.bsonTagFlag = false
			fap.jsonTagFlag = false
			fap.needCounter = false
			fap.AttrsHandler(&res, &auxFunc)
		}
	}
	formRecordsHandler(listener.ShouldImportTime, res, auxFunc)
}

type fieldAttrProcessor struct {
	astListener        *RdtListener
	v                  map[string]any
	tabName, fieldName string
	snakeFieldName     string
	bsonTagFlag        bool
	jsonTagFlag        bool
	needCounter        bool
}

func (f *fieldAttrProcessor) AttrsHandler(
	res *formRec,
	auxFunc *mapset.Set,
) {
	currField := fieldRec{
		f.fieldName: &fieldInfo{
			resolveRDTtype(f.v["ty"].(string)),
			"", "", "",
		},
	}
	fieldAttrs := f.v["attr"].(map[string]any)
	for fieldAttr, maybeStore := range fieldAttrs {
		f.AttrDistributor(res, &currField, fieldAttr, maybeStore)
	}
	currField[f.fieldName].Gtag += fmt.Sprintf(
		"column:%s;", LowerSnakeConvertor(f.fieldName),
	)
	if !f.bsonTagFlag {
		currField[f.fieldName].Bson += fmt.Sprintf(
			"%s,omitempty", f.snakeFieldName,
		)
	}
	if !f.jsonTagFlag {
		currField[f.fieldName].Json += fmt.Sprintf(
			"%s,omitempty", f.snakeFieldName,
		)
	}
	(*res)[f.tabName] = append((*res)[f.tabName], currField)
	if f.needCounter {
		(*auxFunc).Add(f.tabName)
	}
}

func (f *fieldAttrProcessor) AttrDistributor(
	res *formRec, currField *fieldRec,
	fieldAttr string, maybeStore any,
) {
	if fieldAttr == "foreign" {
		f.AttrForeignHandler(
			res,
			maybeStore.(map[string]string),
		)
		(*currField)[f.fieldName].Bson += LowerSnakeConvertor(f.fieldName) + ",omitempty;"
		f.bsonTagFlag, f.jsonTagFlag = true, true

	} else if isAttrForEasiestStructTag(fieldAttr) {
		// primary||notnull||autoincrement||unique
		(*currField)[f.fieldName].Gtag += attrConvertor(fieldAttr)
		if fieldAttr == "primary" {
			primKeyMap[f.tabName] = f.fieldName
		}
	} else if fieldAttr == "createtime" {
		// currField[fieldName].Gtag += "autoCreateTime:" +
		// 	timeUnitConvertor(maybeStore.(string)) + ";"
		extFieldInfo := fieldRec{
			f.fieldName + "CreateAt": &fieldInfo{
				"time.Time",
				"autoCreateTime:" + timeUnitConvertor(maybeStore.(string)) + ";",
				f.snakeFieldName + "_create_at,omitempty",
				f.snakeFieldName + "_create_at,omitempty",
			},
		}
		tsCreator[f.tabName] = f.fieldName + "CreateAt"
		(*res)[f.tabName] = append((*res)[f.tabName], extFieldInfo)
	} else if fieldAttr == "updatetime" {
		// currField[fieldName].Gtag += "autoUpdateTime:" +
		// 	timeUnitConvertor(maybeStore.(string)) + ";"
		extFieldInfo := fieldRec{
			f.fieldName + "UpdateAt": &fieldInfo{
				"time.Time",
				"autoUpdateTime:" + timeUnitConvertor(maybeStore.(string)) + ";",
				f.snakeFieldName + "_update_at,omitempty",
				f.snakeFieldName + "_update_at,omitempty",
			},
		}
		tsUpdater[f.tabName] = f.fieldName + "UpdateAt"
		(*res)[f.tabName] = append((*res)[f.tabName], extFieldInfo)
	} else if fieldAttr == "counter" && !f.needCounter {
		// only permit one counter in a field
		// maintain a counter for current field
		// just add as another member variable
		cntStrVal, ok := maybeStore.(string)
		if !ok {
			panic("can't convert value kept for counter into string")
		}
		cntVal, err := strconv.ParseUint(cntStrVal, 10, 10)
		if err != nil {
			panic("")
		}
		var counterStr = LowerSnakeConvertor("CounterFor" + f.fieldName)
		var extFieldInfo = fieldRec{
			"CounterFor" + f.fieldName: &fieldInfo{
				"uint64", fmt.Sprintf("default:%d", cntVal),
				counterStr + ",omitempty", counterStr + ",omitempty",
			},
		}
		(*res)[f.tabName] = append((*res)[f.tabName], extFieldInfo)
		f.needCounter = true

	} else if fieldAttr == "default" {
		// we need type check for `default`.
		if (*currField)[f.fieldName].Type == "int64" {
			_, err := strconv.ParseInt(maybeStore.(string), 2, 64)
			if err != nil {
				panic(err.Error())
			}
		}
		(*currField)[f.fieldName].Gtag += fmt.Sprintf(
			"default:%s;", maybeStore.(string),
		)
	} else if fieldAttr == "index" {
		(*currField)[f.fieldName].Gtag += fmt.Sprintf(
			"index:%s;", maybeStore.(string),
		)
	} else { /* unknown attribute */
		panic(fmt.Sprintf("encounter an unknown attribute: %s", fieldAttr))
	}
}

// AttrForeignHandler handles for host struct and guest struct, determining how to form
// the foreign-key-relation as the members of structures
func (f *fieldAttrProcessor) AttrForeignHandler(
	res *formRec,
	store map[string]string,
) {
	var refTabName, refFieldName string
	for _k, _v := range store {
		refTabName = _k
		refFieldName = _v
		break // Only one content set in `store`.
	}

	// check whether filling an empty/invalid relation
	_, ok := f.astListener.MetaStruct[refTabName]
	if !ok {
		panic("invalid table name was set up as the origin table foreign key")
	}
	_, ok = f.astListener.MetaStruct[refTabName][refFieldName]
	if !ok {
		panic("invalid field name was provided as foreign key")
	}

	// represent for current structure
	// [NOTE]: sick of GORM... this is insane and unreasonable
	// extFieldName := TopUpperCamelConvertor(ref_tab_name) + "Related"
	extFieldName := f.fieldName + "Related"
	var extFieldInfo = fieldRec{
		extFieldName: &fieldInfo{
			TopUpperCamelConvertor(refTabName), // type
			fmt.Sprintf(
				"foreignKey:%s;references:%s;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;",
				f.fieldName, TopUpperCamelConvertor(refFieldName),
			), "-", "-", // gtag, json and bson
		},
	}

	(*res)[f.tabName] = append((*res)[f.tabName], extFieldInfo)
	outerFormMap[f.tabName] = append(outerFormMap[f.tabName], refTabName)
	// TODO: relation set for package configs?
	// it seems that we can iterate `foreignKeyMap` to complete such goal.

	// For pointee or referred structure, add a new member as well
	refTabName = TopUpperCamelConvertor(refTabName)
	(*res)[refTabName] = append(
		(*res)[refTabName],
		fieldRec{
			// which points back to where the pointer starts
			extFieldName: &fieldInfo{
				strings.Join([]string{"[]", f.tabName}, ""),
				strings.Join([]string{"foreignKey:", f.fieldName}, ""),
				"-", "-",
			},
		},
	)
}

func formRecordsHandler(
	shouldImportTime bool,
	res formRec, auxFunc mapset.Set,
) {
	fmt.Printf(`// Code generated by ./b0gus/rdt-parser/nasty_gen.go; DO NOT EDIT.
// Generated-time: %s
package databases
`,
		time.Now().Format("2006-01-02 15:04:05.000 -0700 MST"),
	)
	if shouldImportTime {
		fmt.Print("import \"time\"\n\n")
	}
	var innerFileHelper = func(name string, attrs *fieldInfo) {
		var (
			tag  = false
			pace = ""
		)
		setTag := func(flip bool) {
			if flip {
				tag = !tag
			}
			if !tag {
				tag = true
				pace = "`"
			} else {
				pace = ""
			}
		}
		fmt.Printf("\t%s %s ", name, attrs.Type)
		if len(attrs.Gtag) > 0 {
			tag = true
			fmt.Printf("`gorm:\"%s\" ", attrs.Gtag)
		}
		if len(attrs.Json) > 0 {
			setTag(false)
			fmt.Printf("%sjson:\"%s\" ", pace, attrs.Json)
		}
		if len(attrs.Bson) > 0 {
			setTag(false)
			fmt.Printf("%sbson:\"%s\" ", pace, attrs.Bson)
		}
		setTag(true)
		fmt.Println(pace)
	}

	for kstructName, fInfos := range res {
		fmt.Printf("type %s struct {\n", kstructName)
		sort.Sort(fInfos) // in dictionary order
		for _, fInfo := range fInfos {
			for fName, fAttrs := range fInfo {
				innerFileHelper(fName, fAttrs)
			}
		}
		fmt.Print("}\n\n")
	}
	for kstructName := range res {
		fmt.Print(FuncAux(res, kstructName, auxFunc))
	}
}

// FuncAux is the code generator.
func FuncAux(
	recTab formRec,
	structureName string,
	inCntSet mapset.Set,
) string {
	tabName := fmt.Sprintf(
		`func (*%s) TableName() string { return "%s" }`+"\n",
		structureName, structureName,
	)
	hasCnt := fmt.Sprintf(
		`func (*%s) HasCounter() bool { return %s }`+"\n",
		structureName,
		strconv.FormatBool(inCntSet.Contains(structureName)),
	)
	// a relation usually requires outer primary key as its managed objects.
	_, ok := primKeyMap[structureName]
	var (
		getPrimKey, PrimKeyName      string
		UpdatedTsName, CreatedTsName string
	)
	if !ok {
		getPrimKey = fmt.Sprintf(
			"func (*%s) GetPrimKey() int64 { return -1 }\n",
			structureName,
		)
		PrimKeyName = fmt.Sprintf(
			"func (*%s) PrimKeyName() string { return \"\" }\n",
			structureName,
		)
	} else {
		getPrimKey = fmt.Sprintf(
			"func (x *%s) GetPrimKey() int64 { return int64(x.%s) }\n",
			structureName, primKeyMap[structureName],
		)
		PrimKeyName = fmt.Sprintf(
			"func (*%s) PrimKeyName() string { return \"%s\" }\n",
			structureName, primKeyMap[structureName],
		)
	}
	_, ok = tsCreator[structureName]
	if !ok {
		CreatedTsName = fmt.Sprintf(
			"func (*%s) CreatedTSname() string { return \"\"}\n",
			structureName,
		)
	} else {
		CreatedTsName = fmt.Sprintf("func (*%s) CreatedTSname() string { return \"%s\"}\n",
			structureName, tsCreator[structureName],
		)
	}
	_, ok = tsUpdater[structureName]
	if !ok {
		UpdatedTsName = fmt.Sprintf(
			"func (*%s) UpdatedTSname() string { return \"\"}\n",
			structureName,
		)
	} else {
		UpdatedTsName = fmt.Sprintf("func (*%s) UpdatedTSname() string { return \"%s\"}\n",
			structureName, tsUpdater[structureName],
		)
	}
	updCnt := fmt.Sprintf(`func (x *%s) UpdateCounter() {`, structureName)
	endlTag := false
	for _, attrInfo := range recTab[structureName] {
		for name := range attrInfo {
			if strings.Contains(name, "CounterFor") {
				endlTag = true
				updCnt += fmt.Sprintf("\n\tx.%s ++", name)
			}
		}
	}
	if endlTag {
		updCnt += "\n"
	}
	updCnt += "}\n"
	return tabName +
		hasCnt + updCnt +
		getPrimKey + PrimKeyName +
		CreatedTsName + UpdatedTsName
}
