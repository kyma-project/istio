export interface DestinationRuleCommands {
    destinationRuleTypeName(value: string): void
    destinationRuleTypeHost(value: string): void
}

Cypress.Commands.add('destinationRuleTypeName', (value: string): void => {
    cy.inputClearAndType('ui5-input[aria-label="DestinationRule name"]', value);
});

Cypress.Commands.add('destinationRuleTypeHost', (value: string): void => {
    cy.inputClearAndType('[data-testid="spec.host"]', value);
});
