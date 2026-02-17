"use client";

import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";

export interface ServerHookValue {
  command: string;
  runOnCreate: boolean;
  runOnDelete: boolean;
  runOnStart: boolean;
  runOnStop: boolean;
  runOnUpdate: boolean;
}

export interface PeerHookValue {
  command: string;
  runOnCreate: boolean;
  runOnDelete: boolean;
  runOnUpdate: boolean;
}

interface ServerHooksEditorProps {
  type: "server";
  value: ServerHookValue[];
  onChange: (hooks: ServerHookValue[]) => void;
}

interface PeerHooksEditorProps {
  type: "peer";
  value: PeerHookValue[];
  onChange: (hooks: PeerHookValue[]) => void;
}

type HooksEditorProps = ServerHooksEditorProps | PeerHooksEditorProps;

export function HooksEditor(props: HooksEditorProps) {
  const { type, value, onChange } = props;

  const addHook = () => {
    if (type === "server") {
      const newHook: ServerHookValue = {
        command: "",
        runOnCreate: false,
        runOnDelete: false,
        runOnStart: false,
        runOnStop: false,
        runOnUpdate: false,
      };
      (onChange as (hooks: ServerHookValue[]) => void)([
        ...(value as ServerHookValue[]),
        newHook,
      ]);
    } else {
      const newHook: PeerHookValue = {
        command: "",
        runOnCreate: false,
        runOnDelete: false,
        runOnUpdate: false,
      };
      (onChange as (hooks: PeerHookValue[]) => void)([
        ...(value as PeerHookValue[]),
        newHook,
      ]);
    }
  };

  const removeHook = (index: number) => {
    const updated = [...value];
    updated.splice(index, 1);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (onChange as (hooks: any[]) => void)(updated);
  };

  const updateHook = (index: number, field: string, val: string | boolean) => {
    const updated = [...value];
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (updated[index] as any)[field] = val;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (onChange as (hooks: any[]) => void)(updated);
  };

  const checkboxFields =
    type === "server"
      ? [
          { key: "runOnCreate", label: "Create" },
          { key: "runOnDelete", label: "Delete" },
          { key: "runOnStart", label: "Start" },
          { key: "runOnStop", label: "Stop" },
          { key: "runOnUpdate", label: "Update" },
        ]
      : [
          { key: "runOnCreate", label: "Create" },
          { key: "runOnDelete", label: "Delete" },
          { key: "runOnUpdate", label: "Update" },
        ];

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-medium">Hooks</Label>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={addHook}
          className="h-7 text-xs"
        >
          <Plus className="mr-1 h-3 w-3" />
          Add Hook
        </Button>
      </div>
      {value.length === 0 && (
        <p className="text-xs text-muted-foreground">
          No hooks configured. Add a hook to run commands on lifecycle events.
        </p>
      )}
      {value.map((hook, index) => (
        <div
          key={index}
          className="flex flex-col gap-2 rounded-md border border-border bg-muted/30 p-3"
        >
          <div className="flex items-center gap-2">
            <Input
              placeholder="Command to execute..."
              className="font-mono text-xs"
              value={hook.command}
              onChange={(e) => updateHook(index, "command", e.target.value)}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
              onClick={() => removeHook(index)}
              aria-label="Remove hook"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
          <div className="flex flex-wrap items-center gap-4">
            <span className="text-xs text-muted-foreground">Run on:</span>
            {checkboxFields.map((field) => (
              <label
                key={field.key}
                className="flex items-center gap-1.5 text-xs"
              >
                <Checkbox
                  checked={
                    (hook as Record<string, unknown>)[field.key] as boolean
                  }
                  onCheckedChange={(checked) =>
                    updateHook(index, field.key, !!checked)
                  }
                />
                {field.label}
              </label>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
