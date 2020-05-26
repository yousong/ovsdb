package types

import (
	"fmt"
	"sort"
	"strings"
)

func (sch *Schema) OrderedTableNames() []string {
	s := make([]string, 0, len(sch.Tables))
	for name, _ := range sch.Tables {
		s = append(s, name)
	}
	sort.Strings(s)
	return s
}

func (sch *Schema) typeName() string {
	name := Kebab2Camel(sch.Name)
	name = ExportName(name)
	for _, tblName := range sch.OrderedTableNames() {
		var (
			tbl        = sch.Tables[tblName]
			tblTypName = tbl.rowTypeName()
		)
		if name == tblTypName {
			return "Ovsdb" + name
		}
	}
	return name
}

func (sch *Schema) gen(w writer) {
	w.Writef(`import "yunion.io/x/ovsdb/types"`)
	w.Writef(`import "github.com/pkg/errors"`)
	w.Writef(`import "fmt"`)

	var schTyp = sch.typeName()
	w.Writef(`type %s struct {`, schTyp)
	for _, name := range sch.OrderedTableNames() {
		var (
			tbl       = sch.Tables[name]
			tblTyp    = tbl.tblTypeName()
			fieldName = tbl.tblField()
		)
		w.Writef(`%s %s`, fieldName, tblTyp)
	}
	w.Writef(`}`)
	w.Writef(`var _ types.IDatabase = &%s{}`, schTyp)
	w.Writef(``)

	w.Writef(`func (db %s) FindOneMatchNonZeros(irow types.IRow) types.IRow {`, schTyp)
	w.Writef(`	switch row := irow.(type) {`)
	for _, name := range sch.OrderedTableNames() {
		var (
			tbl       = sch.Tables[name]
			rowTyp    = tbl.rowTypeName()
			fieldName = tbl.tblField()
		)
		w.Writef(`case *%s:`, rowTyp)
		w.Writef(`	if r := db.%s.FindOneMatchNonZeros(row); r != nil {`, fieldName)
		w.Writef(`		return r`)
		w.Writef(`	}`)
		w.Writef(`	return nil`)
	}
	w.Writef(`	}`)
	w.Writef(`	panic(types.ErrBadType)`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (db %s) FindOneMatchByAnyIndex(irow types.IRow) types.IRow {`, schTyp)
	w.Writef(`	switch row := irow.(type) {`)
	for _, name := range sch.OrderedTableNames() {
		var (
			tbl       = sch.Tables[name]
			rowTyp    = tbl.rowTypeName()
			fieldName = tbl.tblField()
		)
		w.Writef(`case *%s:`, rowTyp)
		w.Writef(`	if r := db.%s.OvsdbGetByAnyIndex(row); r != nil {`, fieldName)
		w.Writef(`		return r`)
		w.Writef(`	}`)
		w.Writef(`	return nil`)
	}
	w.Writef(`	}`)
	w.Writef(`	panic(types.ErrBadType)`)
	w.Writef(`}`)
	w.Writef(``)

	for _, name := range sch.OrderedTableNames() {
		tbl := sch.Tables[name]
		tbl.gen(w)
	}
}

func (tbl *Table) tblTypeName() string {
	name := Kebab2Camel(tbl.Name) + "Table"
	name = ExportName(name)
	return name
}

func (tbl *Table) tblField() string {
	name := Kebab2Camel(tbl.Name)
	name = ExportName(name)
	return name
}

func (tbl *Table) rowTypeName() string {
	name := Kebab2Camel(tbl.Name)
	name = ExportName(name)
	return name
}

// TODO use sync
func (tbl *Table) OrderedColumnNames() []string {
	s := make([]string, 0, len(tbl.Columns))
	for name, _ := range tbl.Columns {
		s = append(s, name)
	}
	sort.Strings(s)
	return s
}

func (tbl *Table) gen(w writer) {
	var (
		tblTyp = tbl.tblTypeName()
		rowTyp = tbl.rowTypeName()
	)

	w.Writef(`type %s []%s`, tblTyp, rowTyp)
	w.Writef(`var _ types.ITable = &%s{}`, tblTyp)
	w.Writef(`func (tbl %s) OvsdbTableName() string {`, tblTyp)
	w.Writef(`	return %q`, tbl.Name)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (tbl %s) OvsdbIsRoot() bool {`, tblTyp)
	w.Writef(`	return %v`, tbl.IsRoot)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (tbl %s) Rows() []types.IRow {`, tblTyp)
	w.Writef(`	r := make([]types.IRow, len(tbl))`)
	w.Writef(`	for i := range tbl {`)
	w.Writef(`		r[i] = &tbl[i]`)
	w.Writef(`	}`)
	w.Writef(`	return r`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (tbl %s) NewRow() types.IRow {`, tblTyp)
	w.Writef(`	return &%s{}`, rowTyp)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (tbl *%s) AppendRow(irow types.IRow) {`, tblTyp)
	w.Writef(`	row := irow.(*%s)`, rowTyp)
	w.Writef(`	*tbl = append(*tbl, *row)`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (tbl %s) OvsdbHasIndex() bool {`, tblTyp)
	w.Writef(`	return %v`, len(tbl.Indexes) > 0)
	w.Writef(`}`)
	w.Writef(``)
	getByFuncNames := make([]string, len(tbl.Indexes))
	for i, index := range tbl.Indexes {
		fields := make([]string, len(index))
		matchSufs := make([]string, len(index))
		for j, colName := range index {
			col := tbl.Columns[colName]
			fields[j] = col.goField()
			matchSufs[j] = col.matchFuncName()
		}
		getByFuncNames[i] = strings.Join(fields, "")

		w.Writef(`func (row *%s) MatchBy%s(row1 *%s) bool {`, rowTyp, getByFuncNames[i], rowTyp)
		for i, field := range fields {
			w.Writef("if !types.%s(row.%s, row1.%s) {", matchSufs[i], field, field)
			w.Writef("	return false")
			w.Writef("}")
		}
		w.Writef(`	return true`)
		w.Writef(`}`)
		w.Writef(``)

		w.Writef(`func (tbl %s) GetBy%s(row1 *%s) *%s {`, tblTyp, getByFuncNames[i], rowTyp, rowTyp)
		w.Writef(`	for i := range tbl {`)
		w.Writef(`		row := &tbl[i]`)
		w.Writef(`		if row.MatchBy%s(row1) {`, getByFuncNames[i])
		w.Writef(`			return row`)
		w.Writef(`		}`)
		w.Writef(`	}`)
		w.Writef(`	return nil`)
		w.Writef(`}`)
		w.Writef(``)
	}
	w.Writef(`func (tbl %s) OvsdbGetByAnyIndex(irow1 types.IRow) types.IRow {`, tblTyp)
	if len(tbl.Indexes) > 0 {
		w.Writef(`	row1 := irow1.(*%s)`, rowTyp)
	}
	for i, index := range tbl.Indexes {
		zeroConds := make([]string, len(index))
		for j, colName := range index {
			col := tbl.Columns[colName]
			zeroConds[j] = fmt.Sprintf("types.%s(row1.%s)", col.isZeroFuncName(), col.goField())
		}
		w.Writef("if !(%s) {", strings.Join(zeroConds, "||"))
		w.Writef("	if row := tbl.GetBy%s(row1); row != nil {", getByFuncNames[i])
		w.Writef("		return row")
		w.Writef("	}")
		w.Writef("}")
	}
	w.Writef(`	return nil`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (tbl %s) FindOneMatchNonZeros(row1 *%s) *%s {`, tblTyp, rowTyp, rowTyp)
	w.Writef(`	for i := range tbl {`)
	w.Writef(`		row := &tbl[i]`)
	w.Writef(`		if row.MatchNonZeros(row1) {`)
	w.Writef(`			return row`)
	w.Writef(`		}`)
	w.Writef(`	}`)
	w.Writef(`	return nil`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`type %s struct {`, rowTyp)
	for _, colName := range tbl.OrderedColumnNames() {
		col := tbl.Columns[colName]
		w.Writef(`%s %s %s`, col.goField(), col.goType(), col.goTags())
	}
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`var _ types.IRow = &%s{}`, rowTyp)

	w.Writef(`func (row *%s) OvsdbTableName() string {`, rowTyp)
	w.Writef(`	return %q`, tbl.Name)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (row *%s) OvsdbIsRoot() bool {`, rowTyp)
	w.Writef(`	return %v`, tbl.IsRoot)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (row *%s) OvsdbUuid() string {`, rowTyp)
	w.Writef(`	return row.Uuid`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (row *%s) OvsdbCmdArgs() []string {`, rowTyp)
	w.Writef(`	r := []string{}`)
	for _, colName := range tbl.OrderedColumnNames() {
		col := tbl.Columns[colName]
		if !col.readOnly() {
			w.Writef(`r = append(r, types.%s(%q, row.%s)...)`, col.cmdArgsFuncName(), colName, col.goField())
		}
	}
	w.Writef(`	return r`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (row *%s) SetColumn(name string, val interface{}) (err error) {`, rowTyp)
	w.Writef(`	defer func() {`)
	w.Writef(`		if panicErr := recover(); panicErr != nil {`)
	w.Writeln(`			err = errors.Wrapf(panicErr.(error), "%s: %#v", name, fmt.Sprintf("%#v", val))`)
	w.Writef(`		}`)
	w.Writef(`	}()`)
	w.Writef(`	switch name {`)
	for _, colName := range tbl.OrderedColumnNames() {
		col := tbl.Columns[colName]
		w.Writef(`case %q:`, colName)
		w.Writef(`	row.%s = types.%s(val)`, col.goField(), col.ensureFuncName())
	}
	w.Writef(`	default:`)
	w.Writef(`		panic(types.ErrUnknownColumn)`)
	w.Writef(`	}`)
	w.Writef(`	return`)
	w.Writef(`}`)
	w.Writef(``)

	w.Writef(`func (row *%s) MatchNonZeros(row1 *%s) bool {`, rowTyp, rowTyp)
	for _, colName := range tbl.OrderedColumnNames() {
		col := tbl.Columns[colName]
		w.Writef(`if !types.%s(row.%s, row1.%s) {`, col.matchNonZeroFuncName(), col.goField(), col.goField())
		w.Writef(`	return false`)
		w.Writef(`}`)
	}
	w.Writef(`	return true`)
	w.Writef(`}`)
	w.Writef(``)

	for _, colName := range tbl.OrderedColumnNames() {
		var (
			col = tbl.Columns[colName]
		)
		if col.Type.Key.Type == Uuid && col.Type.Key.RefTable != "" {
			var (
				refTbl    = tbl.schema.Tables[col.Type.Key.RefTable]
				refRowTyp = refTbl.rowTypeName()
			)
			w.Writef(`func (tbl %s) Find%sReferrer_%s(refUuid string) (r []*%s) {`, tblTyp, refRowTyp, col.Name, tbl.rowTypeName())
			w.Writef(`	for i := range tbl {`)
			w.Writef(`		row := &tbl[i]`)
			ordinal := col.ordinal()
			switch ordinal {
			case ordinalAtom:
				w.Writef(`	if row.%s == refUuid {`, col.goField())
			case ordinalOptional:
				w.Writef(`	if row.%s != nil && *row.%s == refUuid {`, col.goField(), col.goField())
			case ordinalMultiples:
				w.Writef(`	for _, val := range row.%s {`, col.goField())
				w.Writef(`		if val == refUuid {`)
			case ordinalMap:
				w.Writef(`	for val := range row.%s {`, col.goField())
				w.Writef(`		if val == refUuid {`)
			default:
				panic(fmt.Sprintf("table %s column %s: unexpected ordinal %#v",
					tbl.Name, col.Name, ordinal))
			}
			w.Writef(`			r = append(r, row)`)
			w.Writef(`		}`)
			if ordinal == ordinalMultiples || ordinal == ordinalMap {
				w.Writef(`	}`)
			}
			w.Writef(`	}`)
			w.Writef(`	return r`)
			w.Writef(`}`)
			w.Writef(``)
		}
		if col.Type.Value.Type.isValid() &&
			col.Type.Value.Type == Uuid && col.Type.Value.RefTable != "" {
			var (
				refTbl    = tbl.schema.Tables[col.Type.Value.RefTable]
				refRowTyp = refTbl.rowTypeName()
			)
			w.Writef(`func (tbl %s) Find%sReferrer2_%s(refUuid string) (r []*%s) {`, tblTyp, refRowTyp, col.Name, tbl.rowTypeName())
			w.Writef(`	for i := range tbl {`)
			w.Writef(`		row := &tbl[i]`)
			w.Writef(`		for _, val := range row.%s {`, col.goField())
			w.Writef(`			if val == refUuid {`)
			w.Writef(`				r = append(r, row)`)
			w.Writef(`			}`)
			w.Writef(`		}`)
			w.Writef(`	}`)
			w.Writef(`	return r`)
			w.Writef(`}`)
			w.Writef(``)
		}
	}

	{
		_, ok := tbl.Columns["external_ids"]
		g := func(f func()) {
			if ok {
				f()
			} else {
				w.Writef(`	panic(errors.Wrap(types.ErrUnknownColumn, "external_ids"))`)
			}
		}
		w.Writef(`func (row *%s) HasExternalIds() bool {`, rowTyp)
		w.Writef(`	return %v`, ok)
		w.Writef(`}`)
		w.Writef(``)

		w.Writef(`func (row *%s) SetExternalId(k, v string) {`, rowTyp)
		g(func() {
			w.Writef(`	if row.ExternalIds == nil {`)
			w.Writef(`		row.ExternalIds = map[string]string{}`)
			w.Writef(`	}`)
			w.Writef(`	row.ExternalIds[k] = v`)
		})
		w.Writef(`}`)
		w.Writef(``)

		w.Writef(`func (row *%s) GetExternalId(k string) (string, bool) {`, rowTyp)
		g(func() {
			w.Writef(`	if row.ExternalIds == nil {`)
			w.Writef(`		return "", false`)
			w.Writef(`	}`)
			w.Writef(`	r, ok := row.ExternalIds[k]`)
			w.Writef(`	return r, ok`)
		})
		w.Writef(`}`)
		w.Writef(``)

		w.Writef(`func (row *%s) RemoveExternalId(k string) (string, bool) {`, rowTyp)
		g(func() {
			w.Writef(`	if row.ExternalIds == nil {`)
			w.Writef(`		return "", false`)
			w.Writef(`	}`)
			w.Writef(`	r, ok := row.ExternalIds[k]`)
			w.Writef(`	if ok {`)
			w.Writef(`		delete(row.ExternalIds, k)`)
			w.Writef(`	}`)
			w.Writef(`	return r, ok`)
		})
		w.Writef(`}`)
		w.Writef(``)
	}
}

func (col *Column) readOnly() bool {
	name := col.Name
	return name == "_uuid" || name == "_version"
}

func (col *Column) goField() string {
	// TODO
	name := col.Name
	if name == "_uuid" || name == "_version" {
		name = name[1:]
	}
	name = Kebab2Camel(col.Name)
	name = ExportName(name)
	return name
}

func (col *Column) goType() string {
	typ := &col.Type
	kType := typ.Key.Type.goType()

	if typ.Value.Type.isValid() {
		vType := typ.Value.Type.goType()
		return fmt.Sprintf("map[%s]%s", kType, vType)
	}

	if typ.MaxUnlimited {
		return fmt.Sprintf("[]%s", kType)
	}

	min, max := typ.Min, typ.Max
	if min == 0 {
		if max == 0 {
			panic(fmt.Sprintf("column type with min, max both being 0"))
		} else if max == 1 {
			return fmt.Sprintf("*%s", kType)
		} else {
			return fmt.Sprintf("[]%s", kType)
		}
	} else if max == 1 {
		return kType
	} else {
		return fmt.Sprintf("[]%s", kType)
	}
}

func (col *Column) goTags() string {
	return fmt.Sprintf("`json:\"%s\"`", col.Name)
}

type ordinal int

const (
	ordinalAtom ordinal = iota
	ordinalOptional
	ordinalMultiples
	ordinalMap
)

func (col *Column) ordinal() ordinal {
	var (
		typ   = &col.Type
		atomV = typ.Value.Type
	)

	if atomV.isValid() {
		return ordinalMap
	}
	if typ.MaxUnlimited {
		return ordinalMultiples
	}
	min, max := typ.Min, typ.Max
	if min == 0 {
		if max == 0 {
			panic(fmt.Sprintf("column type with min, max both being 0"))
		} else if max == 1 {
			return ordinalOptional
		} else {
			return ordinalMultiples
		}
	} else if max == 1 {
		return ordinalAtom
	} else {
		return ordinalMultiples
	}
}

func (col *Column) funcNameSuffix() string {
	var (
		typ   = &col.Type
		atomK = typ.Key.Type
		atomV = typ.Value.Type
		nameK = atomK.exportName()
	)

	if atomV.isValid() {
		nameV := atomV.exportName()
		return fmt.Sprintf("Map%s%s", nameK, nameV)
	}

	if typ.MaxUnlimited {
		return fmt.Sprintf("%sMultiples", nameK)
	}

	min, max := typ.Min, typ.Max
	if min == 0 {
		if max == 0 {
			panic(fmt.Sprintf("column type with min, max both being 0"))
		} else if max == 1 {
			return fmt.Sprintf("%sOptional", nameK)
		} else {
			return fmt.Sprintf("%sMultiples", nameK)
		}
	} else if max == 1 {
		return fmt.Sprintf("%s", nameK)
	} else {
		return fmt.Sprintf("%sMultiples", nameK)
	}
}

func (col *Column) ensureFuncName() string {
	return "Ensure" + col.funcNameSuffix()
}

func (col *Column) cmdArgsFuncName() string {
	return "OvsdbCmdArgs" + col.funcNameSuffix()
}

func (col *Column) matchNonZeroFuncName() string {
	return "Match" + col.funcNameSuffix() + "IfNonZero"
}

func (col *Column) matchFuncName() string {
	return "Match" + col.funcNameSuffix()
}

func (col *Column) isZeroFuncName() string {
	return "IsZero" + col.funcNameSuffix()
}
