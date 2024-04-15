import 'cypress-file-upload';
import {K8sClientCommands} from "./k8sclient";
import {Status} from "./status";
import {ButtonCommands} from "./buttons";
import {NavigationCommands} from "./navigation";
import {ResourceCommands} from "./resource";

declare global {
    namespace Cypress {
        interface Chainable extends K8sClientCommands, ButtonCommands, NavigationCommands, ResourceCommands {
            loginAndSelectCluster(): void
            clickGenericListLink(resourceName: string): void
            /**
             * Choose an option in a combobox that allows typing. By setting filterFilledComboBoxes to true, the combobox matching the
             * selectors will be filtered to only return elements that have no value, yet.
             * @param selector The selector of the combobox
             * @param optionText The text of the option to choose
             * @param filterFilledComboBoxes If true, only comboboxes that have no value will be considered
             */
            chooseComboboxOption(selector: string, optionText: string, filterFilledComboBoxes?: boolean): void
            /**
             * Choose an option in a combobox that doesn't allow typing
             **/
            chooseComboboxFixedOption(selector: string, optionText: string): void
            filterWithNoValue(): Chainable<JQuery>

            /**
             * Clear an input field and type a new value
             * @param selector The selector of the input field
             * @param newValue The new value to type
             * @param filterFilledInputs If true, only inputs that have no value will be considered
             */
            inputClearAndType(selector: string, newValue: string, filterFilledInputs?: boolean): void
            hasStatusLabel(status: Status): void
            hasTableRowWithLink(hrefValue: string): void
            addFormGroupItem(formGroupSelector: string): void
        }
    }
}