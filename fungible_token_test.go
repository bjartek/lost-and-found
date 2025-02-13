package test_main

import (
	"testing"

	. "github.com/bjartek/overflow"
	"github.com/hexops/autogold"
	"github.com/stretchr/testify/assert"
)

func TestFungibleToken(t *testing.T) {

	otu := NewOverflowTest(t)
	depositAmount := 1000.0
	ExampleTokenVaultType := "A.f8d6e0586b0a20c7.ExampleToken.Vault"

	t.Run("deposit ExampleToken", func(t *testing.T) {

		user := "user1"

		TxWithArgs := otu.O.TxFN(
			WithArg("redeemer", user),
			WithArg("amount", depositAmount),
		)

		signer := WithSigner("account")

		ticketID, err := TxWithArgs("ExampleToken/deposit_example_token",
			signer,
		).
			AssertEvent(t, "TicketDeposited", map[string]interface{}{
				"type": ExampleTokenVaultType,
			}).
			GetIdFromEvent("TicketDeposited", "ticketID")

		assert.NoError(t, err)

		assert.Greater(t, ticketID, uint64(0))

		otu.O.Script("ExampleToken/borrow_ticket_as_struct",
			WithArg("addr", "user1"),
			WithArg("ticketID", ticketID),
		).
			AssertWithPointer(t, "/redeemer", otu.O.Address("user1")).
			AssertWithPointer(t, "/type", ExampleTokenVaultType)

	})

	t.Run("redeem ExampleToken", func(t *testing.T) {

		otu.depositExampleToken("user1", depositAmount)

		otu.O.Tx("ExampleToken/redeem_example_token_all",
			WithSigner("user1"),
		).
			AssertSuccess(t)

		balance, err := otu.O.Script("ExampleToken/get_example_token_balance",
			WithArg("addr", "user1"),
		).
			GetAsInterface()

		assert.NoError(t, err)

		assert.GreaterOrEqual(t, balance.(float64), depositAmount)

		// Shouldn't exist in redeemable type anymore
		otu.O.Script("get_redeemable_types_for_addr",
			WithArg("addr", "user1"),
		).Print().
			AssertWant(t, autogold.Want("This should be nil", nil))

	})

	t.Run("send ExampleToken with setup", func(t *testing.T) {

		otu.O.Tx("ExampleToken/setup_account_ft",
			WithSigner("user1"),
		).
			AssertSuccess(t)

		balance, err := otu.O.Script("ExampleToken/get_example_token_balance",
			WithArg("addr", "user1"),
		).
			GetAsInterface()

		assert.NoError(t, err)

		otu.O.Tx("ExampleToken/try_send_example_token",
			WithSigner("account"),
			WithArg("recipient", "user1"),
			WithArg("amount", depositAmount),
		).
			AssertSuccess(t).
			AssertEvent(t, "ExampleToken.TokensDeposited", map[string]interface{}{
				"amount": depositAmount,
				"to":     otu.O.Address("user1"),
			})

		otu.O.Script("ExampleToken/get_example_token_balance",
			WithArg("addr", "user1"),
		).
			AssertWant(t, autogold.Want("balance should be equal to amount + balance", depositAmount+balance.(float64)))

	})

	t.Run("send ExampleToken without setup", func(t *testing.T) {

		otu.O.Tx("ExampleToken/destroy_example_token_storage",
			WithSigner("user1"),
		).
			AssertSuccess(t)

		_, err := otu.O.Script("ExampleToken/get_example_token_balance",
			WithArg("addr", "user1"),
		).
			GetAsInterface()

		if err != nil {
			assert.Contains(t, err.Error(), "unexpectedly found nil while forcing an Optional value")
		} else {
			assert.Error(t, err)
		}

		otu.O.Tx("ExampleToken/try_send_example_token",
			WithSigner("account"),
			WithArg("recipient", "user1"),
			WithArg("amount", depositAmount),
		).
			AssertSuccess(t).
			AssertEvent(t, "TicketDeposited", map[string]interface{}{
				"type":     ExampleTokenVaultType,
				"redeemer": otu.O.Address("user1"),
			})

	})

	t.Run("should send ExampleToken through Depositor", func(t *testing.T) {
		otu.cleanup("user1").
			setupDepositor("account", 0.0).
			addFlowToDepositor("account", 1.0).
			assertDepositorBalance("account", 1.0)

		otu.O.Tx("ExampleToken/setup_account_ft",
			WithSigner("user1"),
		).
			AssertSuccess(t)

		tokensToSend := 100.0

		otu.O.Tx("ExampleToken/try_send_example_token",
			WithSigner("account"),
			WithArg("recipient", "user1"),
			WithArg("amount", tokensToSend),
		).
			AssertSuccess(t).
			AssertEvent(t, "ExampleToken.TokensDeposited", map[string]interface{}{
				"amount": tokensToSend,
				"to":     otu.O.Address("user1"),
			})

		// assert 0 balance
		otu.O.Script("ExampleToken/get_example_token_balance",
			WithArg("addr", "user1"),
		).
			AssertWant(t, autogold.Want("should be equal to amount sent in", tokensToSend))

	})

	t.Run("should deposit ExampleToken through Depositor", func(t *testing.T) {
		otu.cleanup("user1")

		tokensToSend := 100.0

		otu.O.Tx("ExampleToken/try_send_example_token",
			WithSigner("account"),
			WithArg("recipient", "user1"),
			WithArg("amount", tokensToSend),
		).
			AssertSuccess(t).
			AssertEvent(t, "TicketDeposited", map[string]interface{}{
				"type":     ExampleTokenVaultType,
				"redeemer": otu.O.Address("user1"),
			})

	})

	// New script test
	t.Run("should be able to get total vaults balance to a specific bin and get flow repayment address by script", func(t *testing.T) {
		otu.cleanup("user1")

		tokensToSend := 100.0

		// Send in 100.0 token to user 1's Lost and Found bin
		otu.O.Tx("ExampleToken/try_send_example_token",
			WithSigner("account"),
			WithArg("recipient", "user1"),
			WithArg("amount", tokensToSend),
		).
			AssertSuccess(t).
			AssertEvent(t, "TicketDeposited", map[string]interface{}{
				"type":     ExampleTokenVaultType,
				"redeemer": otu.O.Address("user1"),
			})

		// query the bin's balance. Assert it to be 100
		otu.O.Script("ExampleToken/get_bin_vault_balance",
			WithArg("addr", "user1"),
			WithArg("type", ExampleTokenVaultType),
		).
			AssertWant(t, autogold.Want("should be equal to total sum 100", tokensToSend))

		// Send in 100.0 more token to user 1's Lost and Found bin
		tID, err := otu.O.Tx("ExampleToken/try_send_example_token",
			WithSigner("account"),
			WithArg("recipient", "user1"),
			WithArg("amount", tokensToSend),
		).
			AssertSuccess(t).
			AssertEvent(t, "TicketDeposited", map[string]interface{}{
				"type":     ExampleTokenVaultType,
				"redeemer": otu.O.Address("user1"),
			}).
			GetIdFromEvent("TicketDeposited", "ticketID")

		assert.NoError(t, err)

		// query the bin's balance. Assert it to be 200
		otu.O.Script("ExampleToken/get_bin_vault_balance",
			WithArg("addr", "user1"),
			WithArg("type", ExampleTokenVaultType),
		).
			AssertWant(t, autogold.Want("should be equal to total sum 200", tokensToSend*2))

		// query the bin's balance. Assert it to be 100
		otu.O.Script("get_flowRepayment_address",
			WithArg("addr", "user1"),
			WithArg("type", ExampleTokenVaultType),
			WithArg("ticketID", tID),
		).
			AssertWant(t, autogold.Want("should get admin account address", otu.O.Address("account")))

	})

}
