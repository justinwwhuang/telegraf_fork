package postgresql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/outputs/postgresql/sqltemplate"
	"github.com/influxdata/telegraf/plugins/outputs/postgresql/utils"
)

func TestTableManagerIntegration_EnsureStructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	require.NoError(t, p.Connect())

	cols := []utils.Column{
		p.columnFromTag("foo", ""),
		p.columnFromField("baz", 0),
	}
	missingCols, err := p.tableManager.EnsureStructure(
		ctx,
		p.db,
		p.tableManager.table(t.Name()),
		cols,
		p.CreateTemplates,
		p.AddColumnTemplates,
		p.tableManager.table(t.Name()),
		nil,
	)
	require.NoError(t, err)
	require.Empty(t, missingCols)

	tblCols := p.tableManager.table(t.Name()).columns
	require.EqualValues(t, cols[0], tblCols["foo"])
	require.EqualValues(t, cols[1], tblCols["baz"])
}

func TestTableManagerIntegration_EnsureStructure_alter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	require.NoError(t, p.Connect())

	cols := []utils.Column{
		p.columnFromTag("foo", ""),
		p.columnFromField("bar", 0),
	}
	_, err = p.tableManager.EnsureStructure(
		ctx,
		p.db,
		p.tableManager.table(t.Name()),
		cols,
		p.CreateTemplates,
		p.AddColumnTemplates,
		p.tableManager.table(t.Name()),
		nil,
	)
	require.NoError(t, err)

	cols = append(cols, p.columnFromField("baz", 0))
	missingCols, err := p.tableManager.EnsureStructure(
		ctx,
		p.db,
		p.tableManager.table(t.Name()),
		cols,
		p.CreateTemplates,
		p.AddColumnTemplates,
		p.tableManager.table(t.Name()),
		nil,
	)
	require.NoError(t, err)
	require.Empty(t, missingCols)

	tblCols := p.tableManager.table(t.Name()).columns
	require.EqualValues(t, cols[0], tblCols["foo"])
	require.EqualValues(t, cols[1], tblCols["bar"])
	require.EqualValues(t, cols[2], tblCols["baz"])
}

func TestTableManagerIntegration_EnsureStructure_overflowTableName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	require.NoError(t, p.Connect())

	tbl := p.tableManager.table("ăăăăăăăăăăăăăăăăăăăăăăăăăăăăăăăă") // 32 2-byte unicode characters = 64 bytes
	cols := []utils.Column{
		p.columnFromField("foo", 0),
	}
	_, err = p.tableManager.EnsureStructure(
		ctx,
		p.db,
		tbl,
		cols,
		p.CreateTemplates,
		p.AddColumnTemplates,
		tbl,
		nil,
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "table name too long")
	require.False(t, isTempError(err))
}

func TestTableManagerIntegration_EnsureStructure_overflowTagName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	require.NoError(t, p.Connect())

	tbl := p.tableManager.table(t.Name())
	cols := []utils.Column{
		p.columnFromTag("ăăăăăăăăăăăăăăăăăăăăăăăăăăăăăăăă", "a"), // 32 2-byte unicode characters = 64 bytes
		p.columnFromField("foo", 0),
	}
	_, err = p.tableManager.EnsureStructure(
		ctx,
		p.db,
		tbl,
		cols,
		p.CreateTemplates,
		p.AddColumnTemplates,
		tbl,
		nil,
	)
	require.Error(t, err)
	require.False(t, isTempError(err))
}

func TestTableManagerIntegration_EnsureStructure_overflowFieldName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	require.NoError(t, p.Connect())

	tbl := p.tableManager.table(t.Name())
	cols := []utils.Column{
		p.columnFromField("foo", 0),
		p.columnFromField("ăăăăăăăăăăăăăăăăăăăăăăăăăăăăăăăă", 0),
	}
	missingCols, err := p.tableManager.EnsureStructure(
		ctx,
		p.db,
		tbl,
		cols,
		p.CreateTemplates,
		p.AddColumnTemplates,
		tbl,
		nil,
	)
	require.NoError(t, err)
	require.Len(t, missingCols, 1)
	require.Equal(t, cols[1], missingCols[0])
}

func TestTableManagerIntegration_getColumns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	require.NoError(t, p.Connect())

	cols := []utils.Column{
		p.columnFromTag("foo", ""),
		p.columnFromField("baz", 0),
	}
	_, err = p.tableManager.EnsureStructure(
		ctx,
		p.db,
		p.tableManager.table(t.Name()),
		cols,
		p.CreateTemplates,
		p.AddColumnTemplates,
		p.tableManager.table(t.Name()),
		nil,
	)
	require.NoError(t, err)

	p.tableManager.ClearTableCache()
	require.Empty(t, p.tableManager.table(t.Name()).columns)

	curCols, err := p.tableManager.getColumns(ctx, p.db, t.Name())
	require.NoError(t, err)

	require.EqualValues(t, cols[0], curCols["foo"])
	require.EqualValues(t, cols[1], curCols["baz"])
}

func TestTableManagerIntegration_MatchSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagsAsForeignKeys = true
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]

	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	require.Contains(t, p.tableManager.table(t.Name()+p.TagTableSuffix).columns, "tag")
	require.Contains(t, p.tableManager.table(t.Name()).columns, "a")
}

func TestTableManagerIntegration_MatchSource_UnsignedIntegers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.Uint64Type = PgUint8
	require.NoError(t, p.Init())
	if err := p.Connect(); err != nil {
		if strings.Contains(err.Error(), "retrieving OID for uint8 data type") {
			t.Skipf("pguint extension is not installed")
			t.SkipNow()
		}
		require.NoError(t, err)
	}

	metrics := []telegraf.Metric{
		newMetric(t, "", nil, MSI{"a": uint64(1)}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]

	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	require.Equal(t, PgUint8, p.tableManager.table(t.Name()).columns["a"].Type)
}

func TestTableManagerIntegration_noCreateTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.CreateTemplates = nil
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]

	require.Error(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
}

func TestTableManagerIntegration_noCreateTagTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagTableCreateTemplates = nil
	p.TagsAsForeignKeys = true
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]

	require.Error(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
}

// verify that TableManager updates & caches the DB table structure unless the incoming metric can't fit.
func TestTableManagerIntegration_cache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagsAsForeignKeys = true
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]

	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
}

// Verify that when alter statements are disabled and a metric comes in with a new tag key, that the tag is omitted.
func TestTableManagerIntegration_noAlterMissingTag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.AddColumnTemplates = make([]*sqltemplate.Template, 0)
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))

	metrics = []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 2}),
		newMetric(t, "", MSS{"tag": "foo", "bar": "baz"}, MSI{"a": 3}),
	}
	tsrc = NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	require.NotContains(t, tsrc.ColumnNames(), "bar")
}

// Verify that when using foreign tags and alter statements are disabled and a metric comes in with a new tag key, that
// the tag is omitted.
func TestTableManagerIntegration_noAlterMissingTagTableTag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagsAsForeignKeys = true
	p.TagTableAddColumnTemplates = make([]*sqltemplate.Template, 0)
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))

	metrics = []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 2}),
		newMetric(t, "", MSS{"tag": "foo", "bar": "baz"}, MSI{"a": 3}),
	}
	tsrc = NewTableSources(p.Postgresql, metrics)[t.Name()]
	ttsrc := NewTagTableSource(tsrc)
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	require.NotContains(t, ttsrc.ColumnNames(), "bar")
}

// Verify that when using foreign tags and alter statements generate a permanent error and a metric comes in with a new
// tag key, that the tag is omitted.
func TestTableManagerIntegration_badAlterTagTable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagsAsForeignKeys = true
	tmpl := &sqltemplate.Template{}
	require.NoError(t, tmpl.UnmarshalText([]byte("bad")))
	p.TagTableAddColumnTemplates = []*sqltemplate.Template{tmpl}
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))

	metrics = []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 2}),
		newMetric(t, "", MSS{"tag": "foo", "bar": "baz"}, MSI{"a": 3}),
	}
	tsrc = NewTableSources(p.Postgresql, metrics)[t.Name()]
	ttsrc := NewTagTableSource(tsrc)
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	require.NotContains(t, ttsrc.ColumnNames(), "bar")
}

// verify that when alter statements are disabled and a metric comes in with a new field key, that the field is omitted.
func TestTableManagerIntegration_noAlterMissingField(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.AddColumnTemplates = make([]*sqltemplate.Template, 0)
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))

	metrics = []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 2}),
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 3, "b": 3}),
	}
	tsrc = NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	require.NotContains(t, tsrc.ColumnNames(), "b")
}

// verify that when alter statements generate a permanent error and a metric comes in with a new field key, that the field is omitted.
func TestTableManagerIntegration_badAlterField(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	tmpl := &sqltemplate.Template{}
	require.NoError(t, tmpl.UnmarshalText([]byte("bad")))
	p.AddColumnTemplates = []*sqltemplate.Template{tmpl}
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))

	metrics = []telegraf.Metric{
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 2}),
		newMetric(t, "", MSS{"tag": "foo"}, MSI{"a": 3, "b": 3}),
	}
	tsrc = NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	require.NotContains(t, tsrc.ColumnNames(), "b")
}

func TestTableManager_addColumnTemplates(t *testing.T) {
	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagsAsForeignKeys = true
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"foo": "bar"}, MSI{"a": 1}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))

	p, err = newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagsAsForeignKeys = true
	tmpl := &sqltemplate.Template{}
	require.NoError(t, tmpl.UnmarshalText([]byte(`-- addColumnTemplate: {{ . }}`)))
	p.AddColumnTemplates = []*sqltemplate.Template{tmpl}

	require.NoError(t, p.Connect())

	metrics = []telegraf.Metric{
		newMetric(t, "", MSS{"pop": "tart"}, MSI{"a": 1, "b": 2}),
	}
	tsrc = NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	p.Logger.Info("ok")

	expected := `CREATE TABLE "public"."TestTableManager_addColumnTemplates" ("time" timestamp without time zone, "tag_id" bigint, "a" bigint, "b" bigint)`
	stmtCount := 0
	for _, log := range p.Logger.Logs() {
		if strings.Contains(log.String(), expected) {
			stmtCount++
		}
	}

	require.Equal(t, 1, stmtCount)
}

func TestTableManager_TimeWithTimezone(t *testing.T) {
	p, err := newPostgresqlTest(t)
	require.NoError(t, err)
	p.TagsAsForeignKeys = true
	p.TimestampColumnType = "timestamp with time zone"
	require.NoError(t, p.Init())
	require.NoError(t, p.Connect())

	metrics := []telegraf.Metric{
		newMetric(t, "", MSS{"pop": "tart"}, MSI{"a": 1, "b": 2}),
	}
	tsrc := NewTableSources(p.Postgresql, metrics)[t.Name()]
	require.NoError(t, p.tableManager.MatchSource(ctx, p.db, tsrc))
	p.Logger.Info("ok")

	expected := `CREATE TABLE "public"."TestTableManager_TimeWithTimezone" ("time" timestamp with time zone, "tag_id" bigint, "a" bigint, "b" bigint)`
	stmtCount := 0
	for _, log := range p.Logger.Logs() {
		if strings.Contains(log.String(), expected) {
			stmtCount++
		}
	}

	require.Equal(t, 1, stmtCount)
}
