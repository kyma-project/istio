Cypress.Commands.add('inputClearAndType', (selector: string, newValue: string, filterFilledInputs = false): void => {
    let input = cy.get(selector);

    if (filterFilledInputs) {
        input = input.filterWithNoValue();
    }

    input.scrollIntoView()
        .find('input:visible')
        .click()
        .clear({force: true})
        .type(newValue, {force: true});
});
