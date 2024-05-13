export interface SidecarCommands {
    sidecarTypeName(value: string): void
    sidecarAddIngress(ingress: SidecarIngress): void
}

type SidecarIngress = {
    name: string;
    port: number;
    protocol: "HTTP" | "HTTPS" | "GRPC" | "HTTP2" | "TCP";
    defaultEndpoint: string;
}

Cypress.Commands.add('sidecarTypeName', (value: string): void => {
    cy.inputClearAndType('ui5-input[aria-label="Sidecar name"]', value);
});

Cypress.Commands.add('sidecarAddIngress', (ingress: SidecarIngress): void => {
    cy.addFormGroupItem('[aria-label="expand Ingress"]:visible');
    cy.get('[aria-label="expand Port"]:visible').click();

    cy.inputClearAndType('[data-testid="spec.ingress.0.port.number"]', ingress.port.toString());
    cy.chooseComboboxOption('[data-testid="spec.ingress.0.port.protocol"]', ingress.protocol);
    cy.inputClearAndType('[aria-label="Sidecar name"]', ingress.name, true);
    cy.inputClearAndType('[data-testid="spec.ingress.0.defaultEndpoint"]', ingress.defaultEndpoint);
});
