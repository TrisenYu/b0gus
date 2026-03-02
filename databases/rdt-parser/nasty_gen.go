// / Last modified at 2026/02/11 星期三 22:25:54
package main

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

// convert any string into upper camel form.
func TopUpperCamelConvertor(x string) string {
	x = stringy.New(x).CamelCase().Get()
	if len(x) == 0 {
		panic("an empty string is passed into TopUpperCamelConvertor")
	}
	return strings.ToUpper(x[:1]) + x[1:]
}

// convert any string into upper Snake form.
func fieldNameSnakeConvertor(x string) string {
	return stringy.New(x).SnakeCase().Get()
}

// convert any string into lower Snake form.
func LowerSnakeConvertor(x string) string {
	x = fieldNameSnakeConvertor(x)
	return strings.ToLower(x)
}

// resolve basic RDT datatype.
func resolveRDTtype(x string) string {
	switch x {
	case "text":
		return "string"
	case "integer":
		return "int64"
	default:
		return ""
	}
}

// roughly calculate used bytes of datatype in golang
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

func IsAttrForEasistStructTag(x string) bool {
	return x == "primary" || x == "notnull" ||
		x == "unique" || x == "autoincrement"
}

// convert the string representation of attribute into golang structure tag.
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

// convert time unit for database structure tag in golang.
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

// Requirements:
// 	1. counter for some statstics 				(o)
// 	2. context for autorecoding foreign keys 	(?)
// 	3. generic for all database backends		(?, partially)
// these three requirements should be satisfied at the same time.

type rdtListener struct {
	*BaseRDTparserListener
	MetaStruct       map[string]map[string]any
	ShouldImportTime bool
}

func (r *rdtListener) astAttributesAux(x []IField_attrContext) map[string]any {
	var res = map[string]any{}
	for _, field_attr := range x {
		if field_attr.PRIMARY() != nil {
			res["primary"] = struct{}{}
		} else if field_attr.NOTNULL() != nil {
			res["notnull"] = struct{}{}
		} else if field_attr.UNIQUE() != nil {
			res["unique"] = struct{}{}
		} else if field_attr.AUTOINCREMENT() != nil {
			res["autoincrement"] = struct{}{}
		} else if field_attr.COUNTER() != nil {
			_, ok := res["counter"]
			if ok {
				panic("only allow one counter in one field")
			}
			res["counter"] = field_attr.INT_NUMBER().GetText()
		} else if field_attr.FOREIGN() != nil {
			outer_tab_name := field_attr.Table_name().GetText()
			res["foreign"] = map[string]string{
				outer_tab_name: field_attr.Field_name().GetText(),
			}
		} else if field_attr.CREATETIME() != nil {
			res["createtime"] = field_attr.Time_unit().GetText()
			r.ShouldImportTime = true

		} else if field_attr.UPDATETIME() != nil {
			res["updatetime"] = field_attr.Time_unit().GetText()
			r.ShouldImportTime = true

		} else if field_attr.INDEX() != nil {
			res["index"] = field_attr.Idx_name().GetText()
		} else if field_attr.DEFAULT() != nil {
			if field_attr.Literal() != nil {
				res["default"] = field_attr.Literal().GetText()
			} else {
				res["default"] = field_attr.INT_NUMBER().GetText()
			}
		} else {
			base_str := fmt.Sprintf(
				"unknown token was found: %v",
				field_attr.GetText(),
			)
			panic(base_str)
		}
	}
	return res
}

// hook function for code generation by walking in the AST provided by Antlr4.
func (r *rdtListener) EnterTables(c *TablesContext) {
	tab_name := c.Table_name().GetText()
	r.MetaStruct[tab_name] = map[string]any{}
	for _, field_info := range c.AllAttr_tuples() {
		field_name := field_info.Field_name().GetText()
		var tmp []IField_attrContext
		if field_info.Field_attrs() != nil {
			tmp = field_info.Field_attrs().AllField_attr()
		} else {
			tmp = nil
		}
		r.MetaStruct[tab_name][field_name] = map[string]any{
			"ty":   field_info.Field_type().GetText(),
			"attr": r.astAttributesAux(tmp),
		}
	}
}

func ResolveRDT(fpath string) *rdtListener {
	filestream, err := antlr.NewFileStream(fpath)
	if err != nil {
		return nil
	}
	lexer := NewRDTlexer(filestream)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	parser := NewRDTparser(stream)
	parser.BuildParseTrees = true
	ast := parser.Source_file()
	listener := &rdtListener{}
	listener.MetaStruct = make(map[string]map[string]any)
	walker := antlr.NewParseTreeWalker()
	walker.Walk(listener, ast)
	return listener
}

type (
	fieldInfo struct {
		/* keep this until definition needs extending */
		Type, Gtag, Json, Bson string
	}
	fieldRec map[string]*fieldInfo
	fieldArr []fieldRec
	formRec  map[string]fieldArr
)

func (f fieldArr) Len() int {
	return len(f)
}

func (f fieldArr) Less(i, j int) bool {
	var field1, field2 = f[i], f[j]
	var f1_ty, f2_ty string
	for _, v := range field1 {
		f1_ty = v.Type
	}
	for _, v := range field2 {
		f2_ty = v.Type
	}
	// reverse the order so we get a decresing sequence,
	// thus making the cast of memory alignment lower as much as possible
	return orderNum(f1_ty) > orderNum(f2_ty)
}

func (f fieldArr) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

var (
	// Like a quine, will print itself to another source file.
	foreignKeyMap = map[string][]string{}
	outerFormMap  = map[string][]string{}
	primKeyMap    = map[string]string{}
)

// For host struct and guest struct, this function handles how to form
// the foreign-key-relation as the members of structures
func fieldAttrForeignHandler(
	tab_name, field_name string,
	store map[string]string,
	ast_listener *rdtListener,
	res *formRec,
) {
	var ref_tab_name, ref_field_name string
	for _k, _v := range store {
		ref_tab_name = _k
		ref_field_name = _v
		break // Only one content set in `store`.
	}

	// check whether filling an empty/invalid relation
	_, ok := ast_listener.MetaStruct[ref_tab_name]
	if !ok {
		panic("invalid table name was set up as the origin table foreign key")
	}
	_, ok = ast_listener.MetaStruct[ref_tab_name][ref_field_name]
	if !ok {
		panic("invalid field name was provided as foreign key")
	}

	// represent for current structure
	// [NOTE]: I am sick of gorm... that is insane and unreasonable
	//
	ext_field_name := field_name + "Related"
	// ext_field_name := TopUpperCamelConvertor(ref_tab_name) + "Related"
	var ext_field_info = fieldRec{
		ext_field_name: &fieldInfo{
			TopUpperCamelConvertor(ref_tab_name), // type
			fmt.Sprintf(
				"foreignKey:%s;references:%s;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;",
				field_name, TopUpperCamelConvertor(ref_field_name),
			), "-", "-", // gtag, json and bson
		},
	}

	(*res)[tab_name] = append((*res)[tab_name], ext_field_info)
	foreignKeyMap[tab_name] = append(foreignKeyMap[tab_name], field_name)
	outerFormMap[tab_name] = append(outerFormMap[tab_name], ref_tab_name)
	// TODO: relation set for package configs?
	// it seems that we can iterate `foreignKeyMap` to complete such goal.

	// For pointee or referred structure, add a new member as well
	ref_tab_name = TopUpperCamelConvertor(ref_tab_name)
	(*res)[ref_tab_name] = append(
		(*res)[ref_tab_name],
		fieldRec{
			// which points back to where the pointer starts
			fmt.Sprintf("%s", ext_field_name): &fieldInfo{
				fmt.Sprintf("[]%s", tab_name),
				fmt.Sprintf("foreignKey:%s;", field_name), "-", "-",
			},
		},
	)

}

func fieldAttrsHandler(
	ast_listener *rdtListener,
	tab_name, field_name string,
	v map[string]any,
	res *formRec,
	aux_func *mapset.Set,
) {
	// create variables for further operations
	curr_field := fieldRec{
		field_name: &fieldInfo{
			resolveRDTtype(v["ty"].(string)), "", "", "",
		},
	}
	field_attrs := v["attr"].(map[string]any)
	var (
		bson_tag_flag = false
		json_tag_flag = false
		need_counter  = false
	)
	var snake_field_name = LowerSnakeConvertor(field_name)
	// handle attributes
	for field_attr, maybe_store := range field_attrs {
		if field_attr == "foreign" {
			fieldAttrForeignHandler(
				tab_name, field_name,
				maybe_store.(map[string]string),
				ast_listener, res,
			)
			curr_field[field_name].Bson += LowerSnakeConvertor(field_name) + ",omitempty;"
			bson_tag_flag, json_tag_flag = true, true

		} else if IsAttrForEasistStructTag(field_attr) {
			// primary||notnull||autoincrement||unique
			curr_field[field_name].Gtag += attrConvertor(field_attr)
			if field_attr == "primary" {
				primKeyMap[tab_name] = field_name
			}
		} else if field_attr == "createtime" {
			curr_field[field_name].Gtag += "autoCreateTime:" +
				timeUnitConvertor(maybe_store.(string)) + ";"
			ext_field_info := fieldRec{
				field_name + "CreateAt": &fieldInfo{
					"time.Time", "",
					snake_field_name + "_create_at",
					snake_field_name + "_create_at,omitempty",
				},
			}
			(*res)[tab_name] = append((*res)[tab_name], ext_field_info)
		} else if field_attr == "updatetime" {
			curr_field[field_name].Gtag += "autoUpdateTime:" +
				timeUnitConvertor(maybe_store.(string)) + ";"

			ext_field_info := fieldRec{
				field_name + "UpdateAt": &fieldInfo{
					"time.Time", "",
					snake_field_name + "_update_at",
					snake_field_name + "_update_at,omitempty",
				},
			}
			(*res)[tab_name] = append((*res)[tab_name], ext_field_info)
		} else if field_attr == "counter" && !need_counter {
			// only permit one counter in a field
			// maintain a counter for current field
			// just add as another member variable
			cnt_str_val, ok := maybe_store.(string)
			if !ok {
				panic("can't convert value kept for counter into string")
			}
			cnt_val, err := strconv.ParseUint(cnt_str_val, 10, 10)
			if err != nil {
				panic("")
			}
			var counter_str = LowerSnakeConvertor("CounterFor" + field_name)
			var ext_field_info = fieldRec{
				"CounterFor" + field_name: &fieldInfo{
					"uint64", fmt.Sprintf("default:%d", cnt_val),
					counter_str, counter_str + ",omitempty",
				},
			}
			(*res)[tab_name] = append((*res)[tab_name], ext_field_info)
			need_counter = true

		} else if field_attr == "default" {
			// we need type check for `default`.
			if curr_field[field_name].Type == "int64" {
				_, err := strconv.ParseInt(maybe_store.(string), 2, 64)
				if err != nil {
					panic(err.Error())
				}
			}
			curr_field[field_name].Gtag += fmt.Sprintf(
				"default:%s;", maybe_store.(string),
			)
		} else if field_attr == "index" {
			curr_field[field_name].Gtag += fmt.Sprintf(
				"index:%s;", maybe_store.(string),
			)
		} else { /* unknown attribute */
			panic(fmt.Sprintf("encounter an unknown attribute: %s", field_attr))
		}
	}
	curr_field[field_name].Gtag += fmt.Sprintf(
		"column:%s;", LowerSnakeConvertor(field_name),
	)
	if !bson_tag_flag {
		curr_field[field_name].Bson += fmt.Sprintf(
			"%s,omitempty", snake_field_name,
		)
	}
	if !json_tag_flag {
		curr_field[field_name].Json += fmt.Sprintf(
			"%s,omitempty", snake_field_name,
		)
	}
	(*res)[tab_name] = append((*res)[tab_name], curr_field)
	if need_counter {
		(*aux_func).Add(tab_name)
	}
}

// code generator.
func FuncAux(
	rec_tab formRec,
	structure_name string,
	in_cnt_set mapset.Set,
) string {
	tab_name := fmt.Sprintf(
		`func (*%s) TableName() string { return "%s"; }`+"\n",
		structure_name, structure_name,
	)
	has_cnt := fmt.Sprintf(
		`func (*%s) HasCounter() bool { return %s; }`+"\n",
		structure_name,
		strconv.FormatBool(in_cnt_set.Contains(structure_name)),
	)
	// a relation usually requires outer primary key as its managed objects.
	_, ok := primKeyMap[structure_name]
	var get_prim_key, set_prim_key string
	if !ok {
		get_prim_key = fmt.Sprintf(
			"func (*%s) GetPrimKey() int64 { return -1 }\n",
			structure_name,
		)
	} else {
		get_prim_key = fmt.Sprintf(
			"func (x *%s) GetPrimKey() int64 { return int64(x.%s) }\n",
			structure_name, primKeyMap[structure_name],
		)
	}

	if len(foreignKeyMap[structure_name]) == 0 {
		set_prim_key = fmt.Sprintf(
			"func (x *%s) SetOuterPrimKey(...int64) { return }\n",
			structure_name,
		)
	} else {
		set_prim_key = fmt.Sprintf(
			"func (x *%s) SetOuterPrimKey(arr ...int64) {\n",
			structure_name,
		)
		set_prim_key += fmt.Sprintf(
			`	if len(arr) != %d { panic("unmatch main primary key!") }`+"\n",
			len(foreignKeyMap[structure_name]),
		)
		sort.Strings(foreignKeyMap[structure_name]) // In dictionary order.
		for i, v := range foreignKeyMap[structure_name] {
			set_prim_key += fmt.Sprintf("\tx.%s = arr[%d]\n", v, i)
		}
		set_prim_key += "}\n"
	}

	upd_cnt := fmt.Sprintf(`func (x *%s) UpdateCounter() {`, structure_name)
	endl_tag := false
	for _, attr_info := range rec_tab[structure_name] {
		for name := range attr_info {
			if strings.Contains(name, "CounterFor") {
				endl_tag = true
				upd_cnt += fmt.Sprintf("\n\tx.%s ++", name)
			}
		}
	}
	if endl_tag {
		upd_cnt += "\n"
	}
	upd_cnt += "}\n"
	return tab_name + has_cnt + get_prim_key + set_prim_key + upd_cnt
}

func formRecordsHandler(
	should_import_time bool,
	res formRec,
	aux_func mapset.Set,
) {
	fmt.Printf(`// Code generated by ./B0GUS/rdt-parser/nasty_gen.go; DO NOT EDIT.
// Generated-time: %s
package databases;
`,
		time.Now().Format("2006-01-02 15:04:05.000 -0700 MST"),
	)
	if should_import_time {
		fmt.Printf("import \"time\"\n\n")
	}
	var inline_field_helper = func(name string, attrs *fieldInfo) {
		var (
			tag  bool   = false
			pace string = ""
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

	for kstruct_name, f_infos := range res {
		fmt.Printf("type %s struct {\n", kstruct_name)
		sort.Sort(f_infos)
		for _, f_info := range f_infos {
			for f_name, f_attrs := range f_info {
				inline_field_helper(f_name, f_attrs)
			}
		}
		fmt.Printf("}\n\n")
	}
	for kstruct_name := range res {
		fmt.Print(FuncAux(res, kstruct_name, aux_func))
	}
}

func RDTGenAux(
	listener *rdtListener,
	output_name string,
) {
	fd, err := os.Create(output_name)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"the stage of generating code encounter an error: %v", err,
		)
		return
	}
	os.Stdout = fd
	var (
		res      = formRec{}
		aux_func = mapset.NewSet()
	)
	for tab_name, tab_mems := range listener.MetaStruct {
		tab_name = TopUpperCamelConvertor(tab_name)
		res[tab_name] = make(fieldArr, 0)

		for field_name, v := range tab_mems {
			field_name = TopUpperCamelConvertor(field_name)
			fieldAttrsHandler(
				listener, tab_name, field_name,
				v.(map[string]any), &res, &aux_func,
			)
		}
	}
	formRecordsHandler(listener.ShouldImportTime, res, aux_func)
}

func getRDTfilesInDir(dir string) ([]string, error) {
	var files []string
	aux_fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".rdt" {
			files = append(files, path)
		}
		return nil
	}
	err := filepath.WalkDir(dir, aux_fn)
	return files, err
}

//go:generate go run .
func main() {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		fmt.Fprintf(
			os.Stderr,
			"nasty-gen.main-defer: encounter an error: %v",
			err,
		)
	}()

	src_file_arr, err := getRDTfilesInDir("../rdt-def")
	if err != nil {
		panic(err)
	}
	for _, src_file := range src_file_arr {
		listener := ResolveRDT(src_file)
		if listener == nil {
			continue
		}
		src_file = filepath.Base(src_file)
		split_pos := strings.Index(filepath.Base(src_file), ".")
		switch split_pos {
		case -1:
			split_pos = len(src_file)
		case 0:
			src_file = "aux_gen.go"
			split_pos = len(src_file)
		}
		RDTGenAux(listener, "../"+src_file[:split_pos]+"_gen.go")
	}

	fd, err := os.Create("../../configs/foreign_keymap_gen.go")
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"the stage of generating code encounters an error: %v", err,
		)
		panic(err)
	}
	// For package config;
	os.Stdout = fd
	fmt.Printf(`// Code Generated by ./B0GUS/rdt-parser/nasty_gen.go. DO NOT EDIT.
// Generated-time: %s
package configs

import b0gus_databases "b0gus/databases"

var RecRelationMap = map[string]map[string]any{

`, time.Now().Format("2006-01-02 15:04:05.000 -0700 MST"))
	var def_filter_map = map[string][]string{}
	for tab_name, obj := range outerFormMap {
		// Notice that the one structure might be required in further process.
		for _, item := range obj {
			for _, jtem := range obj {
				if item == jtem {
					continue
				}
				Item, Jtem := TopUpperCamelConvertor(item), TopUpperCamelConvertor(jtem)
				def_filter_map[Item] = append(def_filter_map[Item], Jtem+":"+tab_name)
			}
		}
	}
	for item, obj := range def_filter_map {

		fmt.Printf("\t\"%s\": {\n", item)
		for _, jtem_tab := range obj {
			res := strings.Split(jtem_tab, ":")
			jtem, tab_name := res[0], res[1]
			fmt.Printf("\t\t\"%s\": &b0gus_databases.%s{},\n", jtem, tab_name)
		}
		fmt.Printf("\t},\n")
	}

	fmt.Printf("}")
}
