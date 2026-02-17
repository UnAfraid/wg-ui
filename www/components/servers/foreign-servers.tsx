"use client";

import { useQuery, useMutation } from "@apollo/client";
import { Import, Loader2, Globe } from "lucide-react";
import { toast } from "sonner";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FOREIGN_SERVERS_QUERY, SERVERS_QUERY } from "@/lib/graphql/queries";
import { IMPORT_FOREIGN_SERVER_MUTATION } from "@/lib/graphql/mutations";
import type { ForeignServer } from "@/lib/graphql/types";

export function ForeignServers() {
  const { data, loading } = useQuery(FOREIGN_SERVERS_QUERY);
  const [importServer, { loading: importing }] = useMutation(
    IMPORT_FOREIGN_SERVER_MUTATION,
    {
      refetchQueries: [{ query: SERVERS_QUERY }, { query: FOREIGN_SERVERS_QUERY }],
    }
  );

  const handleImport = async (fs: ForeignServer) => {
    try {
      await importServer({
        variables: { input: { name: fs.name, backendId: fs.backend.id } },
      });
      toast.success(`Server "${fs.name}" imported from ${fs.backend.name}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to import");
    }
  };

  if (loading) return null;

  const foreignServers: ForeignServer[] = data?.foreignServers ?? [];
  if (foreignServers.length === 0) return null;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Globe className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-base">
            Discovered System Interfaces
          </CardTitle>
        </div>
        <p className="text-sm text-muted-foreground">
          WireGuard interfaces found on the system that are not yet managed.
        </p>
      </CardHeader>
      <CardContent>
        <div className="rounded-md border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Backend</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="hidden md:table-cell">Port</TableHead>
                <TableHead className="hidden md:table-cell">Peers</TableHead>
                <TableHead className="w-12">
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {foreignServers.map((fs) => (
                <TableRow key={`${fs.backend.id}-${fs.name}`}>
                  <TableCell className="font-medium">{fs.name}</TableCell>
                  <TableCell>
                    <span className="text-sm text-muted-foreground">
                      {fs.backend.name}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="text-xs">
                      {fs.type}
                    </Badge>
                  </TableCell>
                  <TableCell className="hidden font-mono text-sm text-muted-foreground md:table-cell">
                    {fs.listenPort}
                  </TableCell>
                  <TableCell className="hidden text-sm text-muted-foreground md:table-cell">
                    {fs.peers.length}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={importing}
                      onClick={() => handleImport(fs)}
                    >
                      {importing ? (
                        <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
                      ) : (
                        <Import className="mr-1.5 h-3 w-3" />
                      )}
                      Import
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}
