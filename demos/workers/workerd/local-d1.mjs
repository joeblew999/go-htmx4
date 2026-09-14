// Local-only D1 stand-in for workerd (never deployed).
//
// It implements just the D1 API surface workers-go's database/sql driver uses
// (cloudflare/d1/stmt.go, result.go):
//   db.prepare(sql).bind(...args).run()                     → { meta: { changes, last_row_id } }
//   db.prepare(sql).bind(...args).raw({ columnNames: true }) → [[...columns], ...rows]
// over a Durable Object's SQLite storage, and applies ../migrations on first use.

import { DurableObject } from "cloudflare:workers";
import m0001 from "../migrations/0001_board.sql";

const MIGRATIONS = [["0001_board", m0001]];

export class LocalD1 extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    const sql = ctx.storage.sql;
    ctx.blockConcurrencyWhile(async () => {
      sql.exec("CREATE TABLE IF NOT EXISTS _local_migrations (name TEXT PRIMARY KEY)");
      for (const [name, body] of MIGRATIONS) {
        if (sql.exec("SELECT 1 FROM _local_migrations WHERE name = ?", name).toArray().length) continue;
        sql.exec(body);
        sql.exec("INSERT INTO _local_migrations (name) VALUES (?)", name);
      }
    });
  }

  query(query, params, mode) {
    const sql = this.ctx.storage.sql;
    const cursor = sql.exec(query, ...params);
    const rows = cursor.raw().toArray();
    if (mode === "raw") return [cursor.columnNames, ...rows];
    const { changes, id } = sql.exec("SELECT changes() AS changes, last_insert_rowid() AS id").one();
    return { success: true, results: [], meta: { changes, last_row_id: id } };
  }
}

export function d1Shim(namespace) {
  const db = () => namespace.get(namespace.idFromName("local"));
  return {
    prepare(query) {
      return {
        bind(...params) {
          return {
            run: () => db().query(query, params, "run"),
            raw: () => db().query(query, params, "raw"),
          };
        },
      };
    },
  };
}
