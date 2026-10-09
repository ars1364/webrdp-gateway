"use client";

import { Modal } from "./Modal";
import { Button } from "./ui";

type Props = {
  title: string;
  body: string;
  confirmLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({ title, body, confirmLabel, onConfirm, onCancel }: Props) {
  return (
    <Modal label={title} onClose={onCancel}>
      <h2 className="text-base font-semibold">{title}</h2>
      <p className="mt-2 text-sm text-ink-2">{body}</p>
      <div className="mt-5 flex justify-end gap-2">
        <Button variant="ghost" onClick={onCancel}>Cancel</Button>
        <Button variant="danger" onClick={onConfirm}>{confirmLabel}</Button>
      </div>
    </Modal>
  );
}
