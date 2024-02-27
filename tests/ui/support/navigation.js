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

Cypress.Commands.add('navigateBackTo', (resourceName) => {
    const resourceUrl = resourceName.replace(/\s/g, '').toLowerCase();
    cy.get('ui5-breadcrumbs')
        .find(`ui5-link[href*=${resourceUrl}]`)
        .should('contain.text', resourceName)
        .find(`a[href*=${resourceUrl}]`)
        .click({force: true});
});