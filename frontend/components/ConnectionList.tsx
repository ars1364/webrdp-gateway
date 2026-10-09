"use client";

import type { Connection } from "@/lib/api";
import { Button, Card } from "./ui";

type Props = {
  items: Connection[];
  onConnect: (c: Connection) => void;
  onEdit: (c: Connection) => void;
  onDelete: (c: Connection) => void;
};

export function ConnectionList({ items, onConnect, onEdit, onDelete }: Props) {
  return (
    <Card>
      <h2 className="mb-3 text-base font-semibold">Saved connections</h2>
      {items.length === 0 ? (
        <p className="text-sm text-muted">Nothing saved yet. Fill the form above and press Save.</p>
      ) : (
        <ul className="divide-y divide-line">
          {items.map((c) => (
            <li key={c.id} className="flex flex-wrap items-center gap-3 py-3">
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium" title={c.name}>{c.name}</p>
                <p className="truncate text-sm text-ink-2" title={`${c.host}:${c.port}`}>
                  {c.host}:{c.port}
                  {c.username && <span className="text-muted"> · {c.domain ? `${c.domain}\\` : ""}{c.username}</span>}
                </p>
              </div>
              <Button onClick={() => onConnect(c)}>Connect</Button>
              <Button variant="ghost" onClick={() => onEdit(c)}>Edit</Button>
              <Button variant="danger" onClick={() => onDelete(c)} aria-label={`Delete ${c.name}`}>Delete</Button>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
