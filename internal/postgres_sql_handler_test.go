package foodme

import (
	"fmt"
	"testing"

	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

type ColFilter struct {
	ColumnName  string
	ColumnValue string
	Operator    string
}

type FailingAgent struct{}
type BadFiltersAgent struct{}
type DummyAgent struct {
	Filters       []ColFilter
	JoinFilters   []JoinFilter
	create        bool
	update        bool
	delete        bool
	setCreateFail bool
	setUpdateFail bool
	setDeleteFail bool
}

func (d *DummyAgent) SelectFilters(tableName string, tableAlias string, userInfo map[string]interface{}) (*SelectFilters, error) {
	res := []string{}
	for _, filter := range d.Filters {
		if tableAlias != "" {
			res = append(res, fmt.Sprintf("%s.%s %s %s", tableAlias, filter.ColumnName, filter.Operator, filter.ColumnValue))
		} else {
			res = append(res, fmt.Sprintf("%s %s %s", filter.ColumnName, filter.Operator, filter.ColumnValue))
		}
	}
	jres := []*JoinFilter{}
	for _, filter := range d.JoinFilters {
		jres = append(jres, &JoinFilter{TableName: filter.TableName, Conditions: filter.Conditions})
	}

	return &SelectFilters{WhereFilters: res, JoinFilters: jres}, nil
}

func (d *DummyAgent) CreateAllowed() bool {
	return d.create
}

func (d *DummyAgent) UpdateAllowed() bool {
	return d.update
}

func (d *DummyAgent) DeleteAllowed() bool {
	return d.delete
}

func (d *DummyAgent) SetCreateAllowed(userInfo map[string]interface{}) error {
	if d.setCreateFail {
		return fmt.Errorf("failed to set create allowed")
	}
	return nil
}

func (d *DummyAgent) SetUpdateAllowed(userInfo map[string]interface{}) error {
	if d.setUpdateFail {
		return fmt.Errorf("failed to set update allowed")
	}
	return nil
}

func (d *DummyAgent) SetDeleteAllowed(userInfo map[string]interface{}) error {
	if d.setDeleteFail {
		return fmt.Errorf("failed to set delete allowed")
	}
	return nil
}

func (a *FailingAgent) SelectFilters(tableName string, tableAlias string, userInfo map[string]interface{}) (*SelectFilters, error) {
	return nil, fmt.Errorf("no filters")
}

func (a *FailingAgent) CreateAllowed() bool {
	return false
}

func (a *FailingAgent) UpdateAllowed() bool {
	return false
}

func (a *FailingAgent) DeleteAllowed() bool {
	return false
}

func (a *FailingAgent) SetCreateAllowed(userInfo map[string]interface{}) error {
	return nil
}

func (a *FailingAgent) SetUpdateAllowed(userInfo map[string]interface{}) error {
	return nil
}

func (a *FailingAgent) SetDeleteAllowed(userInfo map[string]interface{}) error {
	return nil
}

func (a *BadFiltersAgent) SelectFilters(tableName string, tableAlias string, userInfo map[string]interface{}) (*SelectFilters, error) {
	return &SelectFilters{WhereFilters: []string{"select * from abhram"}, JoinFilters: []*JoinFilter{}}, nil
}

func (a *BadFiltersAgent) CreateAllowed() bool {
	return false
}

func (a *BadFiltersAgent) UpdateAllowed() bool {
	return false
}

func (a *BadFiltersAgent) DeleteAllowed() bool {
	return false
}

func (a *BadFiltersAgent) SetCreateAllowed(userInfo map[string]interface{}) error {
	return nil
}

func (a *BadFiltersAgent) SetUpdateAllowed(userInfo map[string]interface{}) error {
	return nil
}

func (a *BadFiltersAgent) SetDeleteAllowed(userInfo map[string]interface{}) error {
	return nil
}

func TestHandleSQLWithoutAgent(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	handler := NewPostgresSQLHandler(log, nil)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, sql)
}

func TestHandleBadSQL(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "not a sql statement"
	agent := &DummyAgent{}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.Error(t, err, "at or near \"not\": syntax error")
	assert.Equal(t, res, sql)
}

func TestFailToGetFilters(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	agent := &FailingAgent{}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.Error(t, err, "failed to get filters for table tablename: no filters")
	assert.Equal(t, res, sql)
}

func TestBadFilters(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	agent := &BadFiltersAgent{}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.Error(t, err, "failed to parse where statement for table tablename: at or near \"select\": syntax error")
	assert.Equal(t, res, sql)
}

func TestHandleSimpleSQL(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	agent := &DummyAgent{
		Filters: []ColFilter{
			{ColumnName: "age", ColumnValue: "18", Operator: ">="},
			{ColumnName: "affiliation", ColumnValue: "'royalty'", Operator: "!="},
		},
	}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, "SELECT * FROM tablename WHERE (age >= 18) AND (affiliation != 'royalty')")
}

func TestHandleSimpleJoinSQL(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	agent := &DummyAgent{
		JoinFilters: []JoinFilter{
			{TableName: "othertable", Conditions: "tablename.id = othertable.id AND tablename.secondid = othertable.secondid"},
			{TableName: "thirdtable", Conditions: "tablename.id = thirdtable.id AND tablename.thirdid = thirdtable.thirdid"},
		},
	}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, "SELECT * FROM tablename INNER JOIN othertable ON (tablename.id = othertable.id) AND (tablename.secondid = othertable.secondid) INNER JOIN thirdtable ON (tablename.id = thirdtable.id) AND (tablename.thirdid = thirdtable.thirdid)")

	// bad condition
	agent = &DummyAgent{
		JoinFilters: []JoinFilter{{TableName: "othertable", Conditions: "just a bad condition"}},
	}
	handler = NewPostgresSQLHandler(log, agent)
	_, err = handler.Handle(sql, nil)
	assert.Error(t, err, "failed to parse join statement for table tablename: at or near \"a\": syntax error")

	// Both where and filter conditions together
	agent = &DummyAgent{
		JoinFilters: []JoinFilter{
			{TableName: "othertable", Conditions: "tablename.id = othertable.id AND tablename.secondid = othertable.secondid"},
			{TableName: "thirdtable", Conditions: "tablename.id = thirdtable.id AND tablename.thirdid = thirdtable.thirdid"},
		},
		Filters: []ColFilter{
			{ColumnName: "age", ColumnValue: "18", Operator: ">="},
			{ColumnName: "affiliation", ColumnValue: "'royalty'", Operator: "!="},
		},
	}
	handler = NewPostgresSQLHandler(log, agent)
	res, err = handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, "SELECT * FROM tablename INNER JOIN othertable ON (tablename.id = othertable.id) AND (tablename.secondid = othertable.secondid) INNER JOIN thirdtable ON (tablename.id = thirdtable.id) AND (tablename.thirdid = thirdtable.thirdid) WHERE (age >= 18) AND (affiliation != 'royalty')")
}

func TestHandleAllowSQL(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	agent := &DummyAgent{Filters: []ColFilter{}}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, sql)
}

func TestHandleAdvancedSQL(t *testing.T) {
	log := logrus.StandardLogger()
	agent := &DummyAgent{
		Filters: []ColFilter{
			{ColumnName: "minifield", ColumnValue: "mine", Operator: "="},
		},
	}
	// Unfortunately this has been generated with gpt :(
	sql := `WITH 
-- CTE to get the total hours worked by each employee on all projects
total_hours AS (
    SELECT 
        ep.employee_id, 
        SUM(ep.hours_worked) AS total_hours
    FROM 
        employee_projects ep
    GROUP BY 
        ep.employee_id
),

-- CTE to get the average salary of employees in each department
avg_department_salary AS (
    SELECT 
        e.department_id, 
        AVG(e.salary) AS avg_salary
    FROM 
        employees e
    GROUP BY 
        e.department_id
),

-- CTE to get the latest salary of each employee
latest_salary AS (
    SELECT 
        s.employee_id, 
        MAX(s.salary_date) AS latest_salary_date,
        MAX(s.salary_amount) AS latest_salary_amount
    FROM 
        salaries s
    GROUP BY 
        s.employee_id
),

-- CTE to get the latest bonus of each employee
latest_bonus AS (
    SELECT 
        b.employee_id, 
        MAX(b.bonus_date) AS latest_bonus_date,
        MAX(b.bonus_amount) AS latest_bonus_amount
    FROM 
        bonuses b
    GROUP BY 
        b.employee_id
)

-- Main query to get the details
SELECT 
    e.employee_id, 
    e.first_name, 
    e.last_name, 
    d.department_name, 
    COALESCE(t.total_hours, 0) AS total_hours_worked, 
    COALESCE(l.latest_salary_amount, e.salary) AS current_salary,
    COALESCE(lb.latest_bonus_amount, 0) AS latest_bonus,
    ads.avg_salary AS department_avg_salary,
    (CASE 
        WHEN COALESCE(l.latest_salary_amount, e.salary) > ads.avg_salary THEN 'Above Average'
        WHEN COALESCE(l.latest_salary_amount, e.salary) = ads.avg_salary THEN 'Average'
        ELSE 'Below Average'
    END) AS salary_comparison
FROM 
    employees e
    LEFT JOIN departments d ON e.department_id = d.department_id
    LEFT JOIN total_hours t ON e.employee_id = t.employee_id
    LEFT JOIN latest_salary l ON e.employee_id = l.employee_id
    LEFT JOIN latest_bonus lb ON e.employee_id = lb.employee_id
    LEFT JOIN avg_department_salary ads ON e.department_id = ads.department_id
WHERE 
    EXISTS (
        SELECT 
            1 
        FROM 
            employee_projects ep 
        WHERE 
            ep.employee_id = e.employee_id
    )
    AND NOT EXISTS (
        SELECT 
            1 
        FROM 
            projects p 
        WHERE 
            p.end_date < CURRENT_DATE
    )
UNION
-- Union to get employees without projects but with bonuses
SELECT 
    e.employee_id, 
    e.first_name, 
    e.last_name, 
    d.department_name, 
    0 AS total_hours_worked, 
    COALESCE(l.latest_salary_amount, e.salary) AS current_salary,
    COALESCE(lb.latest_bonus_amount, 0) AS latest_bonus,
    ads.avg_salary AS department_avg_salary,
    (CASE 
        WHEN COALESCE(l.latest_salary_amount, e.salary) > ads.avg_salary THEN 'Above Average'
        WHEN COALESCE(l.latest_salary_amount, e.salary) = ads.avg_salary THEN 'Average'
        ELSE 'Below Average'
    END) AS salary_comparison
FROM 
    employees e
    LEFT JOIN departments d ON e.department_id = d.department_id
    LEFT JOIN latest_salary l ON e.employee_id = l.employee_id
    LEFT JOIN latest_bonus lb ON e.employee_id = lb.employee_id
    LEFT JOIN avg_department_salary ads ON e.department_id = ads.department_id
WHERE 
    NOT EXISTS (
        SELECT 
            1 
        FROM 
            employee_projects ep 
        WHERE 
            ep.employee_id = e.employee_id
    )
    AND EXISTS (
        SELECT 
            1 
        FROM 
            bonuses b 
        WHERE 
            b.employee_id = e.employee_id
    )
ORDER BY 
    e.employee_id;`
	sqlRes := `WITH total_hours AS (SELECT ep.employee_id, sum(ep.hours_worked) AS total_hours FROM employee_projects AS ep WHERE ep.minifield = mine GROUP BY ep.employee_id), avg_department_salary AS (SELECT e.department_id, avg(e.salary) AS avg_salary FROM employees AS e WHERE e.minifield = mine GROUP BY e.department_id), latest_salary AS (SELECT s.employee_id, max(s.salary_date) AS latest_salary_date, max(s.salary_amount) AS latest_salary_amount FROM salaries AS s WHERE s.minifield = mine GROUP BY s.employee_id), latest_bonus AS (SELECT b.employee_id, max(b.bonus_date) AS latest_bonus_date, max(b.bonus_amount) AS latest_bonus_amount FROM bonuses AS b WHERE b.minifield = mine GROUP BY b.employee_id) SELECT e.employee_id, e.first_name, e.last_name, d.department_name, COALESCE(t.total_hours, 0) AS total_hours_worked, COALESCE(l.latest_salary_amount, e.salary) AS current_salary, COALESCE(lb.latest_bonus_amount, 0) AS latest_bonus, ads.avg_salary AS department_avg_salary, (CASE WHEN COALESCE(l.latest_salary_amount, e.salary) > ads.avg_salary THEN 'Above Average' WHEN COALESCE(l.latest_salary_amount, e.salary) = ads.avg_salary THEN 'Average' ELSE 'Below Average' END) AS salary_comparison FROM employees AS e LEFT JOIN departments AS d ON e.department_id = d.department_id LEFT JOIN total_hours AS t ON e.employee_id = t.employee_id LEFT JOIN latest_salary AS l ON e.employee_id = l.employee_id LEFT JOIN latest_bonus AS lb ON e.employee_id = lb.employee_id LEFT JOIN avg_department_salary AS ads ON e.department_id = ads.department_id WHERE (d.minifield = mine) AND ((e.minifield = mine) AND (EXISTS (SELECT 1 FROM employee_projects AS ep WHERE (ep.minifield = mine) AND (ep.employee_id = e.employee_id)) AND (NOT EXISTS (SELECT 1 FROM projects AS p WHERE (p.minifield = mine) AND (p.end_date < current_date()))))) UNION SELECT e.employee_id, e.first_name, e.last_name, d.department_name, 0 AS total_hours_worked, COALESCE(l.latest_salary_amount, e.salary) AS current_salary, COALESCE(lb.latest_bonus_amount, 0) AS latest_bonus, ads.avg_salary AS department_avg_salary, (CASE WHEN COALESCE(l.latest_salary_amount, e.salary) > ads.avg_salary THEN 'Above Average' WHEN COALESCE(l.latest_salary_amount, e.salary) = ads.avg_salary THEN 'Average' ELSE 'Below Average' END) AS salary_comparison FROM employees AS e LEFT JOIN departments AS d ON e.department_id = d.department_id LEFT JOIN latest_salary AS l ON e.employee_id = l.employee_id LEFT JOIN latest_bonus AS lb ON e.employee_id = lb.employee_id LEFT JOIN avg_department_salary AS ads ON e.department_id = ads.department_id WHERE (d.minifield = mine) AND ((e.minifield = mine) AND ((NOT EXISTS (SELECT 1 FROM employee_projects AS ep WHERE (ep.minifield = mine) AND (ep.employee_id = e.employee_id))) AND EXISTS (SELECT 1 FROM bonuses AS b WHERE (b.minifield = mine) AND (b.employee_id = e.employee_id)))) ORDER BY e.employee_id`
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, sqlRes)
}

func TestEmptyTableName(t *testing.T) {
	tbl := &tree.RowsFromExpr{}
	r := getTableNamesAndAliases(tbl)
	assert.Equal(t, len(r), 0)
}

func TestCreate(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "CREATE TABLE test (id INT8)"
	agent := &DummyAgent{Filters: []ColFilter{}}
	handler := NewPostgresSQLHandler(log, agent)
	_, err := handler.Handle(sql, nil)
	assert.Error(t, err, "create operation is not allowed")

	agent.create = true
	handler = NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, sql)
}

func TestUpdate(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "UPDATE test SET age = 18 WHERE name = 'john'"
	agent := &DummyAgent{Filters: []ColFilter{}}
	handler := NewPostgresSQLHandler(log, agent)
	_, err := handler.Handle(sql, nil)
	assert.Error(t, err, "update operation is not allowed")

	agent.update = true
	handler = NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, sql)
}

func TestDelete(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "DROP DATABASE test"
	agent := &DummyAgent{Filters: []ColFilter{}}
	handler := NewPostgresSQLHandler(log, agent)
	_, err := handler.Handle(sql, nil)
	assert.Error(t, err, "delete operation is not allowed")

	agent.delete = true
	handler = NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql, nil)
	assert.NilError(t, err)
	assert.Equal(t, res, sql)
}

func TestSetDDL(t *testing.T) {
	log := logrus.StandardLogger()
	agent := &DummyAgent{Filters: []ColFilter{}}
	handler := NewPostgresSQLHandler(log, agent)

	agent.setCreateFail = true
	err := handler.SetDDL(nil)
	assert.Error(t, err, "failed to set create allowed")

	agent.setCreateFail = false
	agent.setUpdateFail = true
	err = handler.SetDDL(nil)
	assert.Error(t, err, "failed to set update allowed")

	agent.setUpdateFail = false
	agent.setDeleteFail = true
	err = handler.SetDDL(nil)
	assert.Error(t, err, "failed to set delete allowed")

	agent.setDeleteFail = false
	err = handler.SetDDL(nil)
	assert.NilError(t, err)
}
