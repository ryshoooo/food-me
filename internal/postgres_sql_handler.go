package foodme

import (
	"fmt"
	"strings"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/auxten/postgresql-parser/pkg/walk"
	"github.com/sirupsen/logrus"
)

type PostgresSQLHandler struct {
	Logger          *logrus.Logger
	PermissionAgent IPermissionAgent

	withTables   map[string]string
	handleFailed bool
	handleError  error
	userInfo     map[string]interface{}
}

type SelectFilters struct {
	WhereFilters []string
	JoinFilters  []*JoinFilter
}

type JoinFilter struct {
	TableName  string
	Conditions string
}

func NewPostgresSQLHandler(logger *logrus.Logger, pAgent IPermissionAgent) ISQLHandler {
	return &PostgresSQLHandler{Logger: logger, PermissionAgent: pAgent, withTables: make(map[string]string)}
}

func (p *PostgresSQLHandler) Handle(sql string, userInfo map[string]interface{}) (string, error) {
	if p.PermissionAgent == nil {
		return sql, nil
	}

	p.userInfo = userInfo

	statements, err := parser.Parse(sql)
	if err != nil {
		return sql, err
	}

	walker := &walk.AstWalker{Fn: HandleTables}
	_, _ = walker.Walk(statements, p)
	if p.handleFailed {
		return sql, p.handleError
	}
	return statements.String(), nil
}

func HandleTables(ctx interface{}, node interface{}) (stop bool) {
	h := ctx.(*PostgresSQLHandler)
	switch node := node.(type) {
	case *tree.With:
		for _, cte := range node.CTEList {
			h.Logger.Debugf("Found WITH statement aliased as %v", cte.Name.Alias.String())
			h.withTables[cte.Name.Alias.String()] = ""
		}
	case *tree.CreateTable, *tree.CreateChangefeed, *tree.CreateDatabase, *tree.CreateIndex, *tree.CreateRole, *tree.CreateSchema, *tree.CreateSequence, *tree.CreateView, *tree.CreateStats, *tree.CreateStatsOptions:
		if !h.PermissionAgent.CreateAllowed() {
			h.handleFailed = true
			h.handleError = fmt.Errorf("create operation is not allowed")
			return true
		}
	case *tree.Update, *tree.UpdateExpr, *tree.Insert, *tree.AlterIndex, *tree.AlterIndexPartitionBy, *tree.AlterRole, *tree.AlterSequence, *tree.AlterTable:
		if !h.PermissionAgent.UpdateAllowed() {
			h.handleFailed = true
			h.handleError = fmt.Errorf("update operation is not allowed")
			return true
		}
	case *tree.Delete, *tree.DropDatabase, *tree.DropIndex, *tree.DropRole, *tree.DropSequence, *tree.DropTable, *tree.DropView:
		if !h.PermissionAgent.DeleteAllowed() {
			h.handleFailed = true
			h.handleError = fmt.Errorf("delete operation is not allowed")
			return true
		}
	case *tree.SelectClause:
		for _, table := range node.From.Tables {
			tbs := getTableNamesAndAliases(table)
			for _, tb := range tbs {
				if _, ok := h.withTables[tb.TableName]; ok {
					continue
				}
				filters, err := h.PermissionAgent.SelectFilters(tb.TableName, tb.TableAlias, h.userInfo)
				if err != nil {
					h.Logger.Errorf("failed to get filters for table %s: %v", tb.TableName, err)
					h.handleFailed = true
					h.handleError = fmt.Errorf("failed to get filters for table %s: %v", tb.TableName, err)
					return true
				}

				if len(filters.WhereFilters) == 0 && len(filters.JoinFilters) == 0 {
					continue
				}

				if len(filters.WhereFilters) > 0 {
					h.Logger.Debugf("Found where filters for table %s: %v", tb.TableName, filters.WhereFilters)
					swwStmt, err := parser.Parse(fmt.Sprintf("select * from %s where %s", tb.TableName, strings.Join(filters.WhereFilters, " AND ")))
					if err != nil {
						h.Logger.Errorf("failed to parse where statement for table %s: %v", tb.TableName, err)
						h.handleFailed = true
						h.handleError = fmt.Errorf("failed to parse where statement for table %s: %v", tb.TableName, err)
						return true
					}

					whereStatement := swwStmt[0].AST.(*tree.Select).Select.(*tree.SelectClause).Where
					if node.Where == nil {
						node.Where = whereStatement
					} else {
						node.Where.Expr = &tree.AndExpr{Left: whereStatement.Expr, Right: node.Where.Expr}
					}
				}

				if len(filters.JoinFilters) > 0 {
					h.Logger.Debugf("Found join filters for table %s: %v", tb.TableName, filters.JoinFilters)
					h.Logger.Error("join filters are not supported yet, sorry!")
					h.handleFailed = true
					h.handleError = fmt.Errorf("join filters are not supported yet, sorry")
					return true
				}
			}
		}
	}
	return false
}

func (p *PostgresSQLHandler) SetDDL(userInfo map[string]interface{}) error {
	err := p.PermissionAgent.SetCreateAllowed(userInfo)
	if err != nil {
		return err
	}

	err = p.PermissionAgent.SetUpdateAllowed(userInfo)
	if err != nil {
		return err
	}

	err = p.PermissionAgent.SetDeleteAllowed(userInfo)
	if err != nil {
		return err
	}

	return nil
}

type SimpleTable struct {
	TableName  string
	TableAlias string
}

func getTableNamesAndAliases(table tree.TableExpr) []SimpleTable {
	switch tableType := table.(type) {
	case *tree.AliasedTableExpr:
		switch tableExpr := tableType.Expr.(type) {
		case *tree.TableName:
			tableName := tableExpr.TableName.String()
			tableAlias := tableType.As.Alias.String()
			if tableAlias == "\"\"" {
				tableAlias = ""
			}
			return []SimpleTable{{TableName: tableName, TableAlias: tableAlias}}
		}
	case *tree.JoinTableExpr:
		lts := getTableNamesAndAliases(tableType.Left)
		rts := getTableNamesAndAliases(tableType.Right)
		return append(lts, rts...)
	}
	return []SimpleTable{}
}
