Cypress.Commands.add('inputClearAndType', (selector: string, newValue: string, filterFilledInputs = false): void => {
    let input = cy.get(selector)
        .scrollIntoView()
        .find('input:visible');

    if (filterFilledInputs) {
        input = input.filterWithNoValue();
    }

    input.click()
        .clear({force: true})
        .type(newValue, {force: true});
});
