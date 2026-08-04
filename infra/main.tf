terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.100"
    }
  }

  backend "azurerm" {
    resource_group_name  = "orderpulse-tfstate-rg"
    storage_account_name = "orderpulsetf10358"
    container_name       = "tfstate"
    key                  = "orderpulse.tfstate"
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "this" {
  name     = "${var.project_name}-rg"
  location = var.location
}
