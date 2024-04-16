export interface Commands {
    createAuthorizationPolicy(namespace: string, name: string): void
    createService(namespace: string, name: string): void
    createNamespace(name: string): void
    deleteNamespace(name: string): void
}