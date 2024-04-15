import Chainable = Cypress.Chainable;

Cypress.Commands.add('chooseComboboxOption', (selector: string, optionText: string, filterFilledComboBoxes = false) : void => {

    let comboBox = cy.get(`ui5-combobox${selector}:visible`)
        .find('input:visible');

    if (filterFilledComboBoxes) {
        comboBox = comboBox.filterWithNoValue();
    }

    comboBox.click({ force: true })
        .type(optionText, { force: true });

    cy.wait(200);

    cy.get('ui5-li:visible', { timeout: 10000 })
        .contains(optionText)
        .find('li')
        .click({ force: true });
});

Cypress.Commands.add('filterWithNoValue', { prevSubject: true }, (subjects: Chainable<JQuery<HTMLInputElement>>): Chainable<JQuery> => {
    return subjects.filter((_, e) => !(e as HTMLInputElement).value)
});

Cypress.Commands.add('chooseComboboxFixedOption', (selector: string, optionText: string) : void => {

    cy.get(`ui5-combobox${selector}:visible`)
        .find('ui5-icon[accessible-name="Select Options"]:visible')
        .eq(0)
        .click({ force: true });
    cy.wait(200);
    cy.get('ui5-li:visible', { timeout: 10000 })
        .contains(optionText)
        .find('li')
        .click({ force: true });
});
