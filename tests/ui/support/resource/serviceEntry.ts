export interface ServiceEntryCommands {
    serviceEntryTypeName(value: string): void
    serviceEntryTypeHost(value: string): void
    serviceEntrySelectResolution(value: string): void
    serviceEntrySelectLocation(value: string): void
    serviceEntryTypeAddress(value: string): void
}

Cypress.Commands.add('serviceEntryTypeName', (value: string): void => {
    cy.inputClearAndType('ui5-input[aria-label="ServiceEntry name"]', value);
});

Cypress.Commands.add('serviceEntryTypeHost', (value: string): void => {
    cy.get('[aria-label="expand Hosts"]:visible')
        .click();

    cy.inputClearAndType('[data-testid="spec.hosts.0"]:visible', value);
});

Cypress.Commands.add('serviceEntrySelectResolution', (value: string): void => {
    cy.chooseComboboxOption('[data-testid="spec.resolution"]', value);
});

Cypress.Commands.add('serviceEntrySelectLocation', (value: string): void => {
    cy.chooseComboboxOption('[data-testid="spec.location"]', value);
});

Cypress.Commands.add('serviceEntryTypeAddress', (value: string): void => {
    cy.get('[aria-label="expand Addresses"]:visible').click();
    cy.inputClearAndType('[data-testid="spec.addresses.0"]', value);
});

