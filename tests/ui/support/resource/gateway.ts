export interface GatewayCommands {
    gatewayTypeName(value: string): void
    gatewayAddServer(server: GatewayServer): void
}

type GatewayServer = {
    port: number;
    protocol: "HTTP" | "HTTPS" | "GRPC" | "HTTP2" | "TCP";
    name: string;
    host: string;

}

Cypress.Commands.add('gatewayTypeName', (value: string): void => {
    cy.inputClearAndType('ui5-input[aria-label="Gateway name"]', value);
});

Cypress.Commands.add('gatewayAddServer', (server: GatewayServer): void => {
    cy.addFormGroupItem('[aria-label="expand Servers"]:visible');

    cy.inputClearAndType('[data-testid="spec.servers.0.port.number"]', server.port.toString());
    cy.chooseComboboxOption('[data-testid="spec.servers.0.port.protocol"]', server.protocol);

    cy.get('[aria-label^="Gateway name"]:visible')
        .find('input')
        .eq(1)
        .type(server.name, { force: true });

    cy.get('[aria-label="expand Hosts"]:visible').click();
    cy.inputClearAndType('[data-testid="spec.servers.0.hosts.0"]', server.host);
});
