package dbaas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/go-wordwrap"

	exocmd "github.com/exoscale/cli/cmd"
	"github.com/exoscale/cli/pkg/globalstate"
	"github.com/exoscale/cli/pkg/output"
	"github.com/exoscale/cli/table"
	"github.com/exoscale/cli/utils"
	v3 "github.com/exoscale/egoscale/v3"
)

type dbServicePGConnectionPool struct {
	ConnectionURI string `json:"connection_uri"`
	Database      string `json:"database"`
	Mode          string `json:"mode"`
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	Username      string `json:"username"`
}

type dbServicePGComponentShowOutput struct {
	Component string `json:"component"`
	Host      string `json:"host"`
	Port      int64  `json:"port"`
	Route     string `json:"route"`
	Usage     string `json:"usage"`
}

type dbServicePGUserShowOutput struct {
	AllowReplication bool   `json:"allow_replication,omitempty"`
	Password         string `json:"password,omitempty"`
	Type             string `json:"type,omitempty"`
	Username         string `json:"username,omitempty"`
}

type dbServicePGShowOutput struct {
	BackupSchedule  string                           `json:"backup_schedule"`
	Components      []dbServicePGComponentShowOutput `json:"components"`
	Databases       []string                         `json:"databases"`
	ConnectionPools []dbServicePGConnectionPool      `json:"connection_pools"`
	IPFilter        []string                         `json:"ip_filter"`
	URI             string                           `json:"uri"`
	URIParams       map[string]interface{}           `json:"uri_params"`
	Users           []dbServicePGUserShowOutput      `json:"users"`
	Version         string                           `json:"version"`
}

func formatDatabaseServicePGTable(t *table.Table, o *dbServicePGShowOutput) {
	t.Append([]string{"Version", o.Version})
	t.Append([]string{"Backup Schedule", o.BackupSchedule})
	t.Append([]string{"URI", redactDatabaseServiceURI(o.URI)})
	t.Append([]string{"IP Filter", strings.Join(o.IPFilter, ", ")})

	t.Append([]string{"Components", func() string {
		buf := bytes.NewBuffer(nil)
		ct := table.NewEmbeddedTable(buf)
		ct.SetHeader([]string{" "})
		for _, c := range o.Components {
			ct.Append([]string{
				c.Component,
				fmt.Sprintf("%s:%d", c.Host, c.Port),
				"route:" + c.Route,
				"usage:" + c.Usage,
			})
		}
		ct.Render()

		return buf.String()
	}()})

	if len(o.ConnectionPools) > 0 {
		t.Append([]string{"Connection Pools", func() string {
			buf := bytes.NewBuffer(nil)
			pt := table.NewEmbeddedTable(buf)
			pt.SetHeader([]string{" "})
			for _, pool := range o.ConnectionPools {
				pt.Append([]string{
					pool.Name,
					"database:" + pool.Database,
					"size:" + fmt.Sprint(pool.Size),
					"mode:" + pool.Mode,
				})
			}
			pt.Render()

			return buf.String()
		}()})
	}

	t.Append([]string{"Users", func() string {
		if len(o.Users) > 0 {
			return strings.Join(
				func() []string {
					users := make([]string, len(o.Users))
					for i := range o.Users {
						users[i] = fmt.Sprintf("%s (%s)", o.Users[i].Username, o.Users[i].Type)
					}
					return users
				}(),
				"\n")
		}
		return "n/a"
	}()})

	t.Append([]string{"Databases", func() string {
		if len(o.Databases) > 0 {
			return strings.Join(
				func() []string {
					dbs := make([]string, len(o.Databases))
					copy(dbs, o.Databases)
					return dbs
				}(),
				"\n")
		}
		return "n/a"
	}()})

}

func (c *dbaasServiceShowCmd) showDatabaseServicePG(ctx context.Context) (output.Outputter, error) {
	client, err := exocmd.SwitchClientZoneV3(ctx, globalstate.EgoscaleV3Client, v3.ZoneName(c.Zone))
	if err != nil {
		return nil, err
	}

	databaseService, err := client.GetDBAASServicePG(ctx, c.Name)
	if err != nil {
		if errors.Is(err, v3.ErrNotFound) {
			return nil, fmt.Errorf("resource not found in zone %q", c.Zone)
		}
		return nil, err
	}

	switch {
	case c.ShowBackups:
		out := make(dbServiceBackupListOutput, 0)
		if databaseService.Backups != nil {
			for _, b := range databaseService.Backups {
				out = append(out, dbServiceBackupListItemOutput{
					Date: b.BackupTime,
					Name: b.BackupName,
					Size: b.DataSize,
				})
			}
		}
		return &out, nil

	case c.ShowNotifications:
		out := make(dbServiceNotificationListOutput, 0)
		if databaseService.Notifications != nil {
			for _, n := range databaseService.Notifications {
				out = append(out, dbServiceNotificationListItemOutput{
					Level:   string(n.Level),
					Message: wordwrap.WrapString(n.Message, 50),
					Type:    string(n.Type),
				})
			}
		}
		return &out, nil

	case c.ShowSettings != "":
		var out []byte
		switch c.ShowSettings {
		case "pg":
			out, err = json.MarshalIndent(databaseService.PGSettings, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("unable to marshal JSON: %w", err)
			}

		case "pgbouncer":
			out, err = json.MarshalIndent(databaseService.PgbouncerSettings, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("unable to marshal JSON: %w", err)
			}
		case "pglookout":
			out, err = json.MarshalIndent(databaseService.PglookoutSettings, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("unable to marshal JSON: %w", err)
			}
		default:
			return nil, fmt.Errorf(
				"invalid settings value %q, expected one of: %s",
				c.ShowSettings,
				strings.Join(pgSettings, ", "),
			)
		}

		fmt.Println(string(out))
		return nil, nil

	case c.ShowURI:
		// Read password from dedicated endpoint
		client, err := exocmd.SwitchClientZoneV3(
			ctx,
			globalstate.EgoscaleV3Client,
			v3.ZoneName(c.Zone),
		)
		if err != nil {
			return nil, err
		}

		uriParams := databaseService.URIParams

		creds, err := client.RevealDBAASPostgresUserPassword(
			ctx,
			string(databaseService.Name),
			uriParams["user"].(string),
		)
		if err != nil {
			return nil, err
		}

		// Build URI
		uri := fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			uriParams["user"],
			creds.Password,
			uriParams["host"],
			uriParams["port"],
			uriParams["dbname"],
			uriParams["sslmode"],
		)

		fmt.Println(uri)
		return nil, nil
	}

	out := dbServiceShowOutput{
		Zone:                  c.Zone,
		Name:                  string(databaseService.Name),
		Type:                  string(databaseService.Type),
		Plan:                  databaseService.Plan,
		CreationDate:          databaseService.CreatedAT,
		Nodes:                 databaseService.NodeCount,
		NodeCPUs:              databaseService.NodeCPUCount,
		NodeMemory:            databaseService.NodeMemory,
		UpdateDate:            databaseService.UpdatedAT,
		DiskSize:              databaseService.DiskSize,
		State:                 string(databaseService.State),
		TerminationProtection: *databaseService.TerminationProtection,

		Maintenance: func() (v *dbServiceMaintenanceShowOutput) {
			if databaseService.Maintenance != nil {
				v = &dbServiceMaintenanceShowOutput{
					DOW:  string(databaseService.Maintenance.Dow),
					Time: databaseService.Maintenance.Time,
				}
			}
			return
		}(),

		PG: &dbServicePGShowOutput{
			BackupSchedule: func() (v string) {
				if databaseService.BackupSchedule != nil {
					v = fmt.Sprintf(
						"%02d:%02d",
						databaseService.BackupSchedule.BackupHour,
						databaseService.BackupSchedule.BackupMinute,
					)
				}
				return
			}(),

			Components: func() (v []dbServicePGComponentShowOutput) {
				if databaseService.Components != nil {
					for _, c := range databaseService.Components {
						v = append(v, dbServicePGComponentShowOutput{
							Component: c.Component,
							Host:      c.Host,
							Port:      c.Port,
							Route:     string(c.Route),
							Usage:     string(c.Usage),
						})
					}
				}
				return
			}(),

			Databases: func() (v []string) {
				if databaseService.Databases != nil {
					v = make([]string, len(databaseService.Databases))
					for i, d := range databaseService.Databases {
						v[i] = string(d)
					}
				}
				return
			}(),

			ConnectionPools: func() (v []dbServicePGConnectionPool) {
				if databaseService.ConnectionPools != nil {
					for _, pool := range databaseService.ConnectionPools {
						v = append(v, dbServicePGConnectionPool{
							ConnectionURI: pool.ConnectionURI,
							Database:      string(pool.Database),
							Mode:          string(pool.Mode),
							Name:          string(pool.Name),
							Size:          int64(pool.Size),
							Username:      string(pool.Username),
						})
					}
				}
				return
			}(),

			IPFilter: func() (v []string) {
				if databaseService.IPFilter != nil {
					v = databaseService.IPFilter
				}
				return
			}(),

			URI:       databaseService.URI,
			URIParams: databaseService.URIParams,

			Users: func() (v []dbServicePGUserShowOutput) {
				if databaseService.Users != nil {
					for _, u := range databaseService.Users {
						v = append(v, dbServicePGUserShowOutput{
							AllowReplication: utils.DefaultBool(u.AllowReplication, false),
							Password:         u.Password,
							Type:             u.Type,
							Username:         u.Username,
						})
					}
				}
				return
			}(),

			Version: databaseService.Version,
		},
	}

	return &out, nil
}
