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
	Filters []ColFilter
}

func (d *DummyAgent) GetFilters(tableName string, tableAlias string) ([]string, error) {
	res := []string{}
	for _, filter := range d.Filters {
		if tableAlias != "" {
			res = append(res, fmt.Sprintf("%s.%s %s %s", tableAlias, filter.ColumnName, filter.Operator, filter.ColumnValue))
		} else {
			res = append(res, fmt.Sprintf("%s %s %s", filter.ColumnName, filter.Operator, filter.ColumnValue))
		}
	}
	return res, nil
}

func (a *FailingAgent) GetFilters(tableName string, tableAlias string) ([]string, error) {
	return nil, fmt.Errorf("no filters")
}

func (a *BadFiltersAgent) GetFilters(tableName string, tableAlias string) ([]string, error) {
	return []string{"select * from abhram"}, nil
}

func TestHandleSQLWithoutAgent(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	handler := NewPostgresSQLHandler(log, nil)
	res, err := handler.Handle(sql)
	assert.NilError(t, err)
	assert.Equal(t, res, sql)
}

func TestHandleBadSQL(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "not a sql statement"
	agent := &DummyAgent{}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql)
	assert.Error(t, err, "at or near \"not\": syntax error")
	assert.Equal(t, res, sql)
}

func TestFailToGetFilters(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	agent := &FailingAgent{}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql)
	assert.Error(t, err, "failed to get filters for table tablename: no filters")
	assert.Equal(t, res, sql)
}

func TestBadFilters(t *testing.T) {
	log := logrus.StandardLogger()
	sql := "SELECT * FROM tablename"
	agent := &BadFiltersAgent{}
	handler := NewPostgresSQLHandler(log, agent)
	res, err := handler.Handle(sql)
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
	res, err := handler.Handle(sql)
	assert.NilError(t, err)
	assert.Equal(t, res, "SELECT * FROM tablename WHERE (age >= 18) AND (affiliation != 'royalty')")
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
	res, err := handler.Handle(sql)
	assert.NilError(t, err)
	assert.Equal(t, res, sqlRes)
}

func TestEmptyTableName(t *testing.T) {
	tbl := &tree.RowsFromExpr{}
	r := getTableNamesAndAliases(tbl)
	assert.Equal(t, len(r), 0)
}
