resource "azurerm_public_ip" "this" {
  name                = "${var.project_name}-pip"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_network_interface" "this" {
  name                = "${var.project_name}-nic"
  location            = azurerm_resource_group.this.location
  resource_group_name = azurerm_resource_group.this.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = azurerm_subnet.this.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.this.id
  }
}

resource "azurerm_linux_virtual_machine" "this" {
  name                   = "${var.project_name}-vm"
  location               = azurerm_resource_group.this.location
  resource_group_name    = azurerm_resource_group.this.name
  size                   = "Standard_D2as_v5"
  admin_username         = "azureuser"
  network_interface_ids  = [azurerm_network_interface.this.id]

  admin_ssh_key {
    username   = "azureuser"
    public_key = file(pathexpand("~/.ssh/orderpulse_key.pub"))
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
    disk_size_gb         = 30
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  custom_data = filebase64("${path.module}/cloud-init.yaml")
}
