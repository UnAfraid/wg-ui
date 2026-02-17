"use client";

import { useQuery } from "@apollo/client";
import { Link } from "react-router-dom";
import { ArrowLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ServerForm } from "@/components/servers/server-form";
import { SERVER_QUERY } from "@/lib/graphql/queries";
import type { Server } from "@/lib/graphql/types";

export default function EditServerPage({ id }: { id: string }) {
  const { data, loading, error } = useQuery(SERVER_QUERY, {
    variables: { id },
  });

  const server: Server | null = data?.node ?? null;

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }

  if (error || !server) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <p className="text-sm text-muted-foreground">Server not found.</p>
        <Button variant="outline" className="mt-4" asChild>
          <Link to="/servers">Back to Servers</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" className="h-8 w-8" asChild>
          <Link to={`/servers/${id}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            Edit {server.name}
          </h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Update server configuration
          </p>
        </div>
      </div>

      <ServerForm server={server} showAdvanced />
    </div>
  );
}
