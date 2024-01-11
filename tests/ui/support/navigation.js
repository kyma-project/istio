Cypress.Commands.add('navigateTo', (leftNav, resource) => {
    // To check and probably remove after cypress bump
    cy.wait(500);

    cy.getLeftNav()
        .contains(leftNav)

    cy.getLeftNav()
        .contains(leftNav)
        .click({force: true});

    cy.getLeftNav()
        .contains(resource)
        .click();
});

Cypress.Commands.add('getLeftNav', () => {
    return cy.get('aside', { timeout: 10000 });
});