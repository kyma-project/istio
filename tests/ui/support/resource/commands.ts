import {AuthorizationPolicyCommands} from "./authorizationPolicy";
import {DestinationRuleCommands} from "./destinationRule";
import {GatewayCommands} from "./gateway";
import {VirtualServiceCommands} from "./virtualService";
import {ServiceEntryCommands} from "./serviceEntry";
import {SidecarCommands} from "./sidecar";

export interface Commands extends AuthorizationPolicyCommands, DestinationRuleCommands, GatewayCommands,
    VirtualServiceCommands, ServiceEntryCommands, SidecarCommands{
}