"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface CopyableTextProps {
  text: string;
  label?: string;
  truncate?: boolean;
  className?: string;
}

export function CopyableText({
  text,
  label,
  truncate = true,
  className,
}: CopyableTextProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={cn("flex items-center gap-1.5", className)}>
      {label && (
        <span className="text-xs text-muted-foreground">{label}:</span>
      )}
      <code
        className={cn(
          "rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground",
          truncate && "max-w-[200px] truncate"
        )}
        title={text}
      >
        {text}
      </code>
      <Button
        variant="ghost"
        size="icon"
        className="h-6 w-6 shrink-0"
        onClick={handleCopy}
        aria-label={copied ? "Copied" : "Copy to clipboard"}
      >
        {copied ? (
          <Check className="h-3 w-3 text-success" />
        ) : (
          <Copy className="h-3 w-3 text-muted-foreground" />
        )}
      </Button>
    </div>
  );
}
