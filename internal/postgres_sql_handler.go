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
}

func NewPostgresSQLHandler(logger *logrus.Logger, pAgent IPermissionAgent) ISQLHandler {
	return &PostgresSQLHandler{Logger: logger, PermissionAgent: pAgent, withTables: make(map[string]string)}
}

func (p *PostgresSQLHandler) Handle(sql string) (string, error) {
	if p.PermissionAgent == nil {
		return sql, nil
	}

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
	case *tree.SelectClause:
		for _, table := range node.From.Tables {
			tbs := getTableNamesAndAliases(table)
			for _, tb := range tbs {
				if _, ok := h.withTables[tb.TableName]; ok {
					continue
				}
				filters, err := h.PermissionAgent.GetFilters(tb.TableName, tb.TableAlias)
				if err != nil {
					h.Logger.Errorf("failed to get filters for table %s: %v", tb.TableName, err)
					h.handleFailed = true
					h.handleError = fmt.Errorf("failed to get filters for table %s: %v", tb.TableName, err)
					return true
				}

				where := strings.Join(filters, " AND ")
				swwStmt, err := parser.Parse(fmt.Sprintf("select * from %s where %s", tb.TableName, where))
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
		}
	}
	return false
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
